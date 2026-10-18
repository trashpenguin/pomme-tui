use anyhow::Result;
use apple_native_engine::{
    protocol::{WireState, WireTrack},
    queue::Queue,
    transport::{clamp_seek, clamp_volume, validate_resolution},
};
use music::{Equalizer, PlaybackConfig, PlaybackEvent, PlaybackEvents, PlaybackFactory, Player};
use std::{sync::Arc, time::Duration};

pub struct Controller {
    pub queue: Queue,
    pub player: Box<dyn Player>,
    pub events: Box<dyn PlaybackEvents>,
    factory: Arc<dyn PlaybackFactory>,
    config: PlaybackConfig,
    pub generation: u64,
    pub position: u64,
    pub duration: u64,
    pub playing: bool,
    pub paused: bool,
    pub initializing: bool,
    pub ended: bool,
    pub authed: bool,
    pub error: String,
    pub recent: Vec<WireTrack>,
}
impl Controller {
    pub fn save_session(&self) -> Result<()> {
        let Some(root) = std::env::var_os("POMME_DATA_DIR") else {
            return Ok(());
        };
        let root = std::path::PathBuf::from(root);
        std::fs::create_dir_all(&root)?;
        let data = serde_json::json!({"queue":self.queue.items,"index":self.queue.pos,"position":self.position,"volume":(self.config.gain*100.0) as u8,"repeat":self.queue.repeat,"shuffle":self.queue.shuffle});
        let temp = root.join("session.tmp");
        std::fs::write(&temp, serde_json::to_vec(&data)?)?;
        std::fs::rename(temp, root.join("session.json"))?;
        Ok(())
    }
    pub fn restore_session(&mut self) -> Result<()> {
        let Some(root) = std::env::var_os("POMME_DATA_DIR") else {
            return Ok(());
        };
        self.restore_session_path(&std::path::PathBuf::from(root).join("session.json"))
    }
    fn restore_session_path(&mut self, path: &std::path::Path) -> Result<()> {
        if !path.is_file() {
            return Ok(());
        }
        anyhow::ensure!(
            std::fs::metadata(&path)?.len() <= 2_000_000,
            "session too large"
        );
        let saved: serde_json::Value = serde_json::from_slice(&std::fs::read(path)?)?;
        let tracks: Vec<WireTrack> = serde_json::from_value(saved["queue"].clone())?;
        anyhow::ensure!(
            tracks.len() <= 1000
                && tracks
                    .iter()
                    .all(|t| t.kind == "song" && !t.id.is_empty() && t.id.len() <= 256),
            "invalid saved queue"
        );
        self.queue = Queue::new(tracks);
        self.queue.pos = (saved["index"].as_u64().unwrap_or(0) as usize)
            .min(self.queue.items.len().saturating_sub(1));
        self.queue.repeat = saved["repeat"].as_u64().unwrap_or(0).min(2) as u8;
        if saved["shuffle"].as_bool() == Some(true) {
            self.queue.toggle_shuffle()
        }
        self.volume(saved["volume"].as_i64().unwrap_or(50));
        self.duration = self.queue.items.get(self.queue.pos).map_or(0, |t| t.dur_ms);
        self.position = saved["position"].as_u64().unwrap_or(0).min(self.duration);
        self.paused = true;
        self.recover()
    }
    pub fn edit_queue(&mut self, action: &str, index: usize, target: usize) -> Result<()> {
        anyhow::ensure!(
            action == "clear"
                || (index < self.queue.items.len()
                    && (action == "remove"
                        || (action == "move" && target < self.queue.items.len()))),
            "invalid queue edit"
        );
        self.generation = self.generation.wrapping_add(1);
        if action == "clear" {
            self.player.pause();
            self.queue = Queue::default();
            self.playing = false;
            self.paused = true;
            self.initializing = false;
            self.position = 0;
            self.duration = 0;
            let (player, events) = self.factory.start(self.config.clone());
            self.player = player;
            self.events = events;
        } else {
            anyhow::ensure!(index < self.queue.items.len(), "invalid queue index");
            let current = self.queue.pos;
            let mut items = self.queue.items.clone();
            let new_pos = if action == "remove" {
                items.remove(index);
                if current > index {
                    current - 1
                } else {
                    current.min(items.len().saturating_sub(1))
                }
            } else {
                anyhow::ensure!(
                    action == "move" && target < items.len(),
                    "invalid queue destination"
                );
                let item = items.remove(index);
                items.insert(target, item);
                if current == index {
                    target
                } else if index < current && target >= current {
                    current - 1
                } else if index > current && target <= current {
                    current + 1
                } else {
                    current
                }
            };
            let repeat = self.queue.repeat;
            let shuffle = self.queue.shuffle;
            self.queue = Queue::new(items);
            self.queue.pos = new_pos;
            self.queue.repeat = repeat;
            if shuffle {
                self.queue.toggle_shuffle()
            }
            if action == "remove" && index == current {
                if self.queue.items.is_empty() {
                    return self.edit_queue("clear", 0, 0);
                }
                if self.paused {
                    self.position = 0;
                    self.duration = self.queue.items[self.queue.pos].dur_ms;
                    self.recover()?;
                } else {
                    self.start_current()?;
                }
            }
        }
        Ok(())
    }
    pub fn invalidate_auth(&mut self) {
        self.player.pause();
        self.authed = false;
        self.playing = false;
        self.initializing = false;
        self.generation = self.generation.wrapping_add(1);
        self.error = "Apple Music authentication expired; reconnect to sign in".into();
    }
    pub fn refresh_factory(&mut self, factory: Arc<dyn PlaybackFactory>) {
        self.factory = factory;
    }
    pub fn new(factory: Arc<dyn PlaybackFactory>) -> Self {
        let config = PlaybackConfig {
            normalisation: true,
            gapless: false,
            position_interval: Duration::from_millis(250),
            gain: 0.5,
            equalizer: Equalizer::default(),
        };
        let (player, events) = factory.start(config.clone());
        Self {
            queue: Queue::default(),
            player,
            events,
            factory,
            config,
            generation: 0,
            position: 0,
            duration: 0,
            playing: false,
            paused: false,
            initializing: false,
            ended: false,
            authed: true,
            error: String::new(),
            recent: Vec::new(),
        }
    }
    pub fn snapshot(&mut self) -> WireState {
        WireState {
            authed: self.authed,
            playing: self.playing,
            initializing: self.initializing,
            pos_ms: self.position,
            dur_ms: self.duration,
            volume: (self.config.gain * 100.0).round() as u8,
            shuffle: self.queue.shuffle,
            repeat: self.queue.repeat,
            now: self.queue.items.get(self.queue.pos).cloned(),
            queue: self.queue.items.clone(),
            queue_pos: self.queue.pos,
            err: std::mem::take(&mut self.error),
        }
    }
    /// Resolutions commit atomically only after current-generation validation.
    pub fn apply_resolution(&mut self, generation: u64, tracks: Vec<WireTrack>) -> Result<()> {
        validate_resolution(generation, self.generation, &tracks)?;
        let repeat = self.queue.repeat;
        let shuffle = self.queue.shuffle;
        self.queue = Queue::new(tracks);
        self.queue.repeat = repeat;
        if shuffle {
            self.queue.toggle_shuffle();
        }
        self.start_current()
    }
    pub fn insert(&mut self, tracks: Vec<WireTrack>, next: bool) -> Result<()> {
        validate_resolution(0, 0, &tracks)?;
        anyhow::ensure!(
            self.queue.items.len() + tracks.len() <= 1000,
            "queue limit is 1000 tracks"
        );
        let empty = self.queue.items.is_empty();
        if next {
            self.queue.insert_next(tracks);
        } else {
            self.queue.append(tracks);
        }
        if empty {
            self.start_current()?;
        }
        Ok(())
    }
    /// A new event channel isolates even repeated selections of the same catalog ID.
    pub fn start_current(&mut self) -> Result<()> {
        let track = self
            .queue
            .items
            .get(self.queue.pos)
            .cloned()
            .ok_or_else(|| anyhow::anyhow!("queue is empty"))?;
        self.player.pause();
        let (player, events) = self.factory.start(self.config.clone());
        self.player = player;
        self.events = events;
        self.position = 0;
        self.duration = track.dur_ms;
        self.playing = false;
        self.paused = false;
        self.ended = false;
        self.initializing = true;
        self.error.clear();
        self.recent
            .retain(|t| t.id != track.id || t.kind != track.kind);
        self.recent.insert(0, track.clone());
        self.recent.truncate(10);
        self.player.load(&track.id, Duration::ZERO, false)
    }
    pub fn next(&mut self, explicit: bool) -> Result<()> {
        if explicit {
            self.generation = self.generation.wrapping_add(1);
        }
        if self.queue.advance(explicit).is_some() {
            self.start_current()?;
        } else {
            self.player.pause();
            self.playing = false;
            self.initializing = false;
            self.ended = true;
        }
        Ok(())
    }
    pub fn previous(&mut self) -> Result<()> {
        self.generation = self.generation.wrapping_add(1);
        if self.position > 3000 {
            self.seek(0);
        } else if self.queue.previous().is_some() {
            self.start_current()?;
        } else {
            self.seek(0);
        }
        Ok(())
    }
    pub fn jump(&mut self, index: usize) -> Result<()> {
        self.queue.jump(index)?;
        self.generation = self.generation.wrapping_add(1);
        self.start_current()
    }
    pub fn play_pause(&mut self) -> Result<()> {
        anyhow::ensure!(!self.queue.items.is_empty(), "queue is empty");
        if self.ended {
            return self.start_current();
        }
        self.paused = !self.paused;
        self.playing = !self.paused && !self.initializing;
        if self.paused {
            self.player.pause();
        } else {
            self.player.play();
        }
        Ok(())
    }
    pub fn seek(&mut self, position: u64) {
        self.position = clamp_seek(position, self.duration);
        self.player.seek(Duration::from_millis(self.position));
    }
    pub fn volume(&mut self, volume: i64) {
        self.config.gain = f32::from(clamp_volume(volume)) / 100.0;
        self.player.set_gain(self.config.gain);
    }
    pub fn recover(&mut self) -> Result<()> {
        let Some(track) = self.queue.items.get(self.queue.pos) else {
            return Ok(());
        };
        self.player.pause();
        let (player, events) = self.factory.start(self.config.clone());
        self.player = player;
        self.events = events;
        self.playing = false;
        self.initializing = !self.paused && !self.ended;
        if self.paused || self.ended {
            self.player
                .load_paused_at(&track.id, Duration::from_millis(self.position))?;
        } else {
            self.player
                .load(&track.id, Duration::from_millis(self.position), false)?;
        }
        Ok(())
    }
    pub fn spectrum(&self) -> [f32; 32] {
        let mut output = [0.0; 32];
        if !self.playing || self.paused {
            return output;
        }
        if let Some(spectrum) = self.player.spectrum() {
            for (to, from) in output.iter_mut().zip(spectrum.bands()) {
                *to = if from.is_finite() {
                    from.clamp(0.0, 1.0)
                } else {
                    0.0
                };
            }
        }
        output
    }
    pub fn event(&mut self, event: PlaybackEvent) -> Result<()> {
        if let Some(id) = event.id() {
            if self
                .queue
                .items
                .get(self.queue.pos)
                .is_none_or(|t| t.id != id)
            {
                return Ok(());
            }
        }
        match event {
            PlaybackEvent::Playing { at, .. } => {
                self.initializing = false;
                self.position = at.as_millis() as u64;
                if self.paused {
                    self.player.pause();
                    self.playing = false;
                } else {
                    self.playing = true;
                }
            }
            PlaybackEvent::Paused { at, .. } => {
                self.position = at.as_millis() as u64;
                self.playing = false;
                self.initializing = false;
            }
            PlaybackEvent::Position { at, .. } | PlaybackEvent::Seeked { at, .. } => {
                self.position = at.as_millis() as u64
            }
            PlaybackEvent::Length { duration, .. } => self.duration = duration.as_millis() as u64,
            PlaybackEvent::Ended { .. } => {
                self.position = self.duration;
                self.next(false)?;
            }
            PlaybackEvent::OutputChanged => self.recover()?,
            PlaybackEvent::Unavailable { .. }
            | PlaybackEvent::Throttled { .. }
            | PlaybackEvent::Refused
            | PlaybackEvent::Gated => {
                self.player.pause();
                self.playing = false;
                self.initializing = false;
                self.error = "Apple Music declined playback; try another item or reconnect".into();
            }
            _ => {}
        }
        Ok(())
    }
}
impl Drop for Controller {
    fn drop(&mut self) {
        self.player.pause();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Mutex;
    struct FakeFactory(Arc<Mutex<Vec<String>>>);
    struct FakePlayer(Arc<Mutex<Vec<String>>>);
    struct FakeEvents;
    #[async_trait::async_trait]
    impl PlaybackEvents for FakeEvents {
        async fn next(&mut self) -> Option<PlaybackEvent> {
            std::future::pending().await
        }
    }
    impl Player for FakePlayer {
        fn load(&self, id: &str, at: Duration, _: bool) -> Result<()> {
            self.0
                .lock()
                .unwrap()
                .push(format!("play:{id}:{}", at.as_millis()));
            Ok(())
        }
        fn load_paused_at(&self, id: &str, at: Duration) -> Result<()> {
            self.0
                .lock()
                .unwrap()
                .push(format!("paused:{id}:{}", at.as_millis()));
            Ok(())
        }
        fn preload(&self, _: &str, _: bool) -> Result<()> {
            Ok(())
        }
        fn play(&self) {
            self.0.lock().unwrap().push("resume".into());
        }
        fn pause(&self) {
            self.0.lock().unwrap().push("pause".into());
        }
        fn seek(&self, _: Duration) {}
        fn set_gain(&self, gain: f32) {
            self.0.lock().unwrap().push(format!("gain:{gain}"));
        }
    }
    impl PlaybackFactory for FakeFactory {
        fn start(&self, config: PlaybackConfig) -> (Box<dyn Player>, Box<dyn PlaybackEvents>) {
            self.0
                .lock()
                .unwrap()
                .push(format!("start:{}", config.gain));
            (Box::new(FakePlayer(self.0.clone())), Box::new(FakeEvents))
        }
    }
    fn controller() -> (Controller, Arc<Mutex<Vec<String>>>) {
        let log = Arc::new(Mutex::new(Vec::new()));
        let mut ctrl = Controller::new(Arc::new(FakeFactory(log.clone())));
        ctrl.apply_resolution(0, vec![track("a"), track("b")])
            .unwrap();
        (ctrl, log)
    }
    fn track(id: &str) -> WireTrack {
        WireTrack {
            id: id.into(),
            kind: "song".into(),
            dur_ms: 20000,
            ..Default::default()
        }
    }
    #[test]
    fn paused_recovery_is_ready_and_space_can_resume() {
        let (mut ctrl, log) = controller();
        ctrl.paused = true;
        ctrl.position = 12000;
        ctrl.volume(35);
        ctrl.recover().unwrap();
        assert!(
            !ctrl.initializing,
            "paused recovery must not await an audible sample"
        );
        assert!(!ctrl.playing);
        assert_eq!(ctrl.position, 12000);
        assert!(log.lock().unwrap().contains(&"start:0.35".into()));
        assert!(log.lock().unwrap().contains(&"paused:a:12000".into()));
        ctrl.play_pause().unwrap();
        assert!(!ctrl.paused);
        assert!(log.lock().unwrap().contains(&"resume".into()));
    }
    #[test]
    fn explicit_selections_supersede_delayed_play_resolution() {
        for action in ["jump", "next", "prev"] {
            let (mut ctrl, _) = controller();
            ctrl.generation = 1;
            match action {
                "jump" => ctrl.jump(1).unwrap(),
                "next" => ctrl.next(true).unwrap(),
                _ => ctrl.previous().unwrap(),
            }
            assert!(
                ctrl.apply_resolution(1, vec![track("late")]).is_err(),
                "{action} did not cancel old selection"
            );
            assert_ne!(ctrl.queue.items[0].id, "late");
        }
    }
    #[test]
    fn invalid_jump_does_not_supersede_active_selection() {
        let (mut ctrl, _) = controller();
        ctrl.generation = 4;
        assert!(ctrl.jump(99).is_err());
        assert_eq!(ctrl.generation, 4);
    }
    #[test]
    fn auth_invalidation_stops_playback_and_rejects_pending_selection() {
        let (mut ctrl, log) = controller();
        ctrl.playing = true;
        ctrl.initializing = true;
        ctrl.generation = 2;
        ctrl.invalidate_auth();
        assert!(!ctrl.authed);
        assert!(!ctrl.playing);
        assert!(!ctrl.initializing);
        assert!(ctrl.apply_resolution(2, vec![track("late")]).is_err());
        assert!(log.lock().unwrap().contains(&"pause".into()));
    }
    #[test]
    fn editing_queue_keeps_current_track_and_resets_indices() {
        let (mut ctrl, _) = controller();
        ctrl.edit_queue("move", 0, 1).unwrap();
        assert_eq!(ctrl.queue.items[ctrl.queue.pos].id, "a");
        ctrl.edit_queue("remove", 0, 0).unwrap();
        assert_eq!(ctrl.queue.items[ctrl.queue.pos].id, "a");
        ctrl.edit_queue("clear", 0, 0).unwrap();
        assert!(ctrl.queue.items.is_empty());
        assert!(!ctrl.playing && !ctrl.initializing);
    }

    #[test]
    fn restored_queue_is_paused_at_saved_position() {
        let (mut ctrl, log) = controller();
        let path =
            std::env::temp_dir().join(format!("pomme-session-test-{}.json", fastrand::u64(..)));
        std::fs::write(&path,serde_json::to_vec(&serde_json::json!({"queue":[track("a"),track("b")],"index":1,"position":10000,"volume":25,"repeat":2,"shuffle":false})).unwrap()).unwrap();
        let result = ctrl.restore_session_path(&path);
        std::fs::remove_file(path).unwrap();
        result.unwrap();
        assert!(ctrl.paused && !ctrl.playing && !ctrl.initializing);
        assert_eq!(ctrl.queue.pos, 1);
        assert_eq!(ctrl.position, 10000);
        assert!(log.lock().unwrap().contains(&"paused:b:10000".into()));
    }
}
