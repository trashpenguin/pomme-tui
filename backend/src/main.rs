mod catalog;
mod media_keys;
mod controller;
mod signin;
use anyhow::{Context, Result, bail};
use apple_native_engine::auth_failure::is_auth_error;
use apple_native_engine::protocol::{self, Request, Response, SearchResults, WireTrack};
use controller::Controller;
use music::{MusicProvider, SignIn};
use serde_json::{Value, json};
use std::{
    collections::{BTreeMap, HashSet},
    io::{BufRead, Write},
    sync::Arc,
    time::Duration,
};

enum Input {
    Request(Request),
    Invalid(u64, String),
    Eof,
}
enum Data {
    Results(SearchResults),
    Tracks(Vec<WireTrack>),
}
struct Completed {
    id: u64,
    method: String,
    generation: u64,
    sequence: u64,
    result: Result<Data>,
}

/// Reads one bounded JSON line, draining an oversized frame without retaining its contents.
fn read_frame(reader: &mut impl BufRead) -> std::io::Result<Option<Vec<u8>>> {
    let mut result = Vec::new();
    let mut oversized = false;
    loop {
        let buf = reader.fill_buf()?;
        if buf.is_empty() {
            return if result.is_empty() {
                Ok(None)
            } else {
                Ok(Some(result))
            };
        }
        let newline = buf.iter().position(|&b| b == b'\n');
        let length = newline.map_or(buf.len(), |n| n + 1);
        if result.len() + length > protocol::MAX_LINE {
            oversized = true;
        }
        if !oversized {
            result.extend_from_slice(&buf[..length]);
        }
        reader.consume(length);
        if newline.is_some() {
            return if oversized {
                Err(std::io::Error::new(
                    std::io::ErrorKind::InvalidData,
                    "request exceeds size limit",
                ))
            } else {
                Ok(Some(result))
            };
        }
    }
}
fn write_response(response: Response) -> Result<()> {
    let mut output = std::io::stdout().lock();
    serde_json::to_writer(&mut output, &response)?;
    writeln!(output)?;
    output.flush()?;
    Ok(())
}
fn string<'a>(params: &'a Value, key: &str) -> Result<&'a str> {
    params
        .get(key)
        .and_then(Value::as_str)
        .with_context(|| format!("missing string parameter {key}"))
}
fn number(params: &Value, key: &str) -> Result<u64> {
    params
        .get(key)
        .and_then(Value::as_u64)
        .with_context(|| format!("invalid nonnegative parameter {key}"))
}

#[tokio::main]
async fn main() -> Result<()> {
    if let Some(module) = widevine::hosted() {
        return widevine::host(&module);
    }
    let provider = Arc::new(music::apple::AppleProvider::new());
    if std::env::args().nth(1).as_deref() == Some("login") {
        let canceled = if std::env::args().any(|arg| arg == "--owned-stdin") {
            apple_native_engine::owner::monitor_owner(std::io::stdin())
        } else {
            Arc::new(std::sync::atomic::AtomicBool::new(false))
        };
        let cookies = signin::login_cookie(&*provider, &canceled)?;
        let (tx, rx) = tokio::sync::mpsc::unbounded_channel();
        tx.send(cookies)?;
        let exchange = tokio::time::timeout(
            Duration::from_secs(45),
            provider.sign_in(SignIn::Secret, Arc::new(|_| {}), rx),
        );
        tokio::pin!(exchange);
        loop {
            tokio::select! {result=&mut exchange=>{result??;break;},_=tokio::time::sleep(Duration::from_millis(50))=>{
                anyhow::ensure!(!canceled.load(std::sync::atomic::Ordering::Acquire),"Sign-in owner closed");
            }}
        }
        eprintln!("Apple Music sign-in saved locally");
        return Ok(());
    }
    let (tx, mut incoming) = tokio::sync::mpsc::channel(64);
    std::thread::spawn(move || {
        let mut input = std::io::stdin().lock();
        loop {
            match read_frame(&mut input) {
                Ok(Some(bytes)) => {
                    let item = match protocol::decode(&bytes) {
                        Ok(request) => Input::Request(request),
                        Err(error) => {
                            let id = serde_json::from_slice::<Value>(&bytes)
                                .ok()
                                .and_then(|v| v.get("id").and_then(Value::as_u64))
                                .unwrap_or(0);
                            Input::Invalid(id, error.to_string())
                        }
                    };
                    if tx.blocking_send(item).is_err() {
                        break;
                    }
                }
                Ok(None) => {
                    let _ = tx.blocking_send(Input::Eof);
                    break;
                }
                Err(error) => {
                    if tx
                        .blocking_send(Input::Invalid(0, "invalid or oversized input frame".into()))
                        .is_err()
                    {
                        break;
                    }
                    if error.kind() != std::io::ErrorKind::InvalidData {
                        let _ = tx.blocking_send(Input::Eof);
                        break;
                    }
                }
            }
        }
    });
    let (media_tx,mut media_rx)=tokio::sync::mpsc::unbounded_channel();
    let mut media:Option<media_keys::Bridge>=None;
    let mut media_tick=tokio::time::interval(Duration::from_millis(500));
    let mut session = None;
    let mut control: Option<Controller> = None;
    let mut catalog = None;
    let mut jobs = tokio::task::JoinSet::<Completed>::new();
    let mut auth_checks =
        tokio::task::JoinSet::<(u64, Result<Option<music::ProviderSession>>)>::new();
    let mut active = HashSet::new();
    let mut queued = BTreeMap::new();
    let mut queue_sequence = 0;
    let mut next_queue = 1;
    let mut save_tick=tokio::time::interval(Duration::from_secs(15));
    loop {
        if control.as_ref().is_some_and(|ctrl| !ctrl.authed) {
            jobs.abort_all();
            auth_checks.abort_all();
            queued.clear();
            next_queue = queue_sequence + 1;
            for id in active.drain() {
                write_response(Response::error(
                    id,
                    "Apple Music authentication expired; sign in again",
                ))?;
            }
        }
        tokio::select! {
            _=media_tick.tick()=>{if let (Some(bridge),Some(ctrl))=(media.as_ref(),control.as_ref()){
                let track=ctrl.queue.items.get(ctrl.queue.pos);
                bridge.update(media_keys::Display{title:track.map_or(String::new(),|t|t.title.clone()),artist:track.map_or(String::new(),|t|t.artist.clone()),playing:ctrl.playing,enabled:ctrl.authed && track.is_some()});
            }},
            Some(action)=media_rx.recv()=>{if let Some(ctrl)=control.as_mut(){if ctrl.authed {let result=match action {
                media_keys::Action::Play=>{if ctrl.paused || ctrl.ended {ctrl.play_pause()}else{Ok(())}},
                media_keys::Action::Pause=>{if !ctrl.paused && !ctrl.queue.items.is_empty(){ctrl.play_pause()}else{Ok(())}},
                media_keys::Action::Next=>ctrl.next(true),media_keys::Action::Previous=>ctrl.previous()
            };if result.is_err(){ctrl.error="Media-key action could not complete".into();}}}},
            _=save_tick.tick()=>{if let Some(ctrl)=control.as_ref(){if ctrl.authed{let _=ctrl.save_session();}}},
            _=tokio::signal::ctrl_c()=>break,
            input=incoming.recv()=>{
                let request=match input {
                    Some(Input::Request(request))=>request,
                    Some(Input::Invalid(id,error))=>{write_response(Response::error(id,error))?;continue;},
                    _=>break,
                };
                let id=request.id;let method=request.method.as_str();
                if active.contains(&id){write_response(Response::error(id,"duplicate active request ID"))?;continue;}
                if method=="shutdown"{if let Some(ctrl)=control.as_ref(){let _=ctrl.save_session();}write_response(Response::ok(id,true))?;break;}
                if method=="hello" {
                    if control.is_none(){
                        match tokio::time::timeout(Duration::from_secs(45),provider.restore()).await {
                            Ok(Ok(Some(restored)))=>{
                                if widevine::require().is_err(){write_response(Response::error(id,"Widevine is unavailable. Start Pomme.exe, or run pomme doctor; install Google Chrome x64 if missing."))?;continue;}
                                catalog=Some(catalog::Catalog{api:restored.api.clone()});let mut restored_control=Controller::new(restored.playback.clone());
                                if restored_control.restore_session().is_err(){restored_control.error="Saved session could not be restored".into();}
                                if media.is_none(){match media_keys::Bridge::start(media_tx.clone()){Ok(bridge)=>media=Some(bridge),Err(_)=>restored_control.error="Windows media controls unavailable".into()}}
                                control=Some(restored_control);session=Some(restored);
                            },
                            Ok(Ok(None))=>{},
                            _=>{write_response(Response::error(id,"Apple Music session could not be restored; sign in again"))?;continue;},
                        }
                    }
                    write_response(Response::ok(id,json!({"protocol":1,"authed":control.as_ref().is_some_and(|c|c.authed)})))?;continue;
                }
                let Some(ctrl)=control.as_mut() else{write_response(Response::error(id,"Apple Music sign-in required"))?;continue;};
                if !ctrl.authed && method!="state" {write_response(Response::error(id,"Apple Music authentication expired; sign in again"))?;continue;}
                if ["search","library","library_page","artist_page","save_playlist","play","queue_next","queue_later"].contains(&method){
                    if jobs.len()>=8{write_response(Response::error(id,"too many catalog requests; try again"))?;continue;}
                    if method=="play"{ctrl.generation=ctrl.generation.wrapping_add(1);}
                    let generation=ctrl.generation;let cat=catalog.as_ref().unwrap().clone();
                    let sequence=if method=="queue_next"||method=="queue_later"{queue_sequence+=1;queue_sequence}else{0};
                    let saved_queue=if method=="save_playlist" {ctrl.queue.items.clone()} else {Vec::new()};
                    active.insert(id);
                    jobs.spawn(async move {
                        let result=tokio::time::timeout(Duration::from_secs(if request.method=="save_playlist" {300} else {30}),async {
                            match request.method.as_str(){
                                "save_playlist"=>cat.save_queue(string(&request.params,"name")?,&saved_queue).await.map(Data::Results),
                                "search"=>cat.search(string(&request.params,"query")?).await.map(Data::Results),
                                "library_page"=>cat.library_page(string(&request.params,"category")?,number(&request.params,"offset")? as usize).await.map(Data::Results),
                                "artist_page"=>cat.artist_page(string(&request.params,"id")?).await.map(Data::Results),
                                "library"=>cat.library().await.map(Data::Results),
                                _=>cat.resolve(string(&request.params,"kind")?,string(&request.params,"id")?).await.map(Data::Tracks),
                            }
                        }).await.unwrap_or_else(|_|Err(anyhow::anyhow!("request timed out; a playlist save may have completed partially")));
                        Completed{id,method:request.method,generation,sequence,result}
                    });continue;
                }
                let result:Result<Value>=(||{
                    match method {
                        "queue_edit"=>{ctrl.edit_queue(string(&request.params,"action")?, request.params["index"].as_u64().unwrap_or(0) as usize, request.params["target"].as_u64().unwrap_or(0) as usize)?;Ok(json!(true))},
                        "state"=>Ok(serde_json::to_value(ctrl.snapshot())?),
                        "spectrum"=>Ok(json!(ctrl.spectrum())),
                        "pause"=>{if !ctrl.paused && !ctrl.queue.items.is_empty(){ctrl.play_pause()?;}Ok(json!(true))},
                        "play_pause"=>{ctrl.play_pause()?;Ok(json!(true))},
                        "next"=>{ctrl.next(true)?;Ok(json!(true))},
                        "prev"=>{ctrl.previous()?;Ok(json!(true))},
                        "jump"=>{ctrl.jump(number(&request.params,"index")? as usize)?;Ok(json!(true))},
                        "seek"=>{ctrl.seek(number(&request.params,"atMs")?);Ok(json!(true))},
                        "set_volume"=>{ctrl.volume(request.params.get("volume").and_then(Value::as_i64).context("invalid volume")?);Ok(json!(true))},
                        "toggle_shuffle"=>{ctrl.queue.toggle_shuffle();Ok(json!(true))},
                        "cycle_repeat"=>{ctrl.queue.cycle_repeat();Ok(json!(true))},
                        "reload"=>{ctrl.recover()?;Ok(json!(true))},
                        "account"=>Ok(json!({"name":session.as_ref().map(|s|s.profile.display_name.clone()).unwrap_or_default(),"handle":"","country":provider.location().unwrap_or_default(),"storefront":""})),
                        "sign_out"=>{let _=ctrl.edit_queue("clear",0,0);ctrl.invalidate_auth();provider.sign_out();Ok(json!(true))},
                        _=>bail!("unknown method"),
                    }
                })();
                write_response(match result{Ok(value)=>Response::ok(id,value),Err(_)=>Response::error(id,"invalid request parameters or unavailable operation")})?;
            }
            event=async{match control.as_mut(){Some(ctrl)=>ctrl.events.next().await,None=>std::future::pending().await}}=>{
                match event{Some(event)=>{if let Some(ctrl)=control.as_mut(){
                    let declined=matches!(&event,music::PlaybackEvent::Unavailable{..}|music::PlaybackEvent::Refused|music::PlaybackEvent::Gated);
                    if ctrl.event(event).is_err(){ctrl.error="Audio output could not be restored".into();ctrl.playing=false;ctrl.initializing=false;}
                    if declined && ctrl.authed && auth_checks.is_empty(){let provider=provider.clone();let generation=ctrl.generation;
                        auth_checks.spawn(async move{(generation,tokio::time::timeout(Duration::from_secs(45),provider.restore()).await.unwrap_or_else(|_|Err(anyhow::anyhow!("account check timed out"))))});}
                }},None=>break}
            }
            checked=auth_checks.join_next(),if !auth_checks.is_empty()=>{
                if let Some(Ok((generation,result)))=checked {if let Some(ctrl)=control.as_mut(){if generation==ctrl.generation && ctrl.authed {
                    match result {
                        Ok(Some(restored))=>{ctrl.refresh_factory(restored.playback.clone());catalog=Some(catalog::Catalog{api:restored.api.clone()});session=Some(restored);},
                        Ok(None)=>ctrl.invalidate_auth(),Err(error) if is_auth_error(&error)=>ctrl.invalidate_auth(),_=>{},
                    }
                }}}
            }
            completed=jobs.join_next(),if !jobs.is_empty()=>{
                let Some(Ok(completed))=completed else{continue;};
                if completed.sequence!=0 {
                    queued.insert(completed.sequence,completed);
                    while let Some(completed)=queued.remove(&next_queue){next_queue+=1;active.remove(&completed.id);finish(completed,control.as_mut().unwrap())?;}
                }else{active.remove(&completed.id);finish(completed,control.as_mut().unwrap())?;}
            }
        }
    }
    jobs.abort_all();
    if let Some(ctrl)=control.as_ref(){let _=ctrl.save_session();}
    drop(control);
    Ok(())
}
fn finish(completed: Completed, control: &mut Controller) -> Result<()> {
    if !control.authed {
        return write_response(Response::error(
            completed.id,
            "Apple Music authentication expired; sign in again",
        ));
    }
    let response = match completed.result {
        Ok(Data::Results(mut result)) => {
            if completed.method == "library" {
                let mut recent = control.recent.clone();
                for track in result.recent {
                    if !recent
                        .iter()
                        .any(|t| t.id == track.id && t.kind == track.kind)
                    {
                        recent.push(track);
                    }
                }
                recent.truncate(10);
                result.recent = recent;
            }
            Response::ok(completed.id, result)
        }
        Ok(Data::Tracks(tracks)) => {
            let applied = if completed.method == "play" {
                control.apply_resolution(completed.generation, tracks)
            } else {
                if completed.generation != control.generation {
                    Err(anyhow::anyhow!("queue selection superseded"))
                } else {
                    control.insert(tracks, completed.method == "queue_next")
                }
            };
            match applied {
                Ok(()) => Response::ok(completed.id, true),
                Err(error) => Response::error(completed.id, error.to_string()),
            }
        }
        Err(error) => {
            if is_auth_error(&error) {
                control.invalidate_auth();
                Response::error(
                    completed.id,
                    "Apple Music authentication expired; sign in again",
                )
            } else {
                Response::error(
                    completed.id,
                    "Catalog item or request is unavailable; try again",
                )
            }
        }
    };
    write_response(response)
}
