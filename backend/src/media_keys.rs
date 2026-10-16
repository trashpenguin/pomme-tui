use std::{sync::mpsc, time::Duration};
use windows::{
    core::{w, HSTRING},
    Foundation::TypedEventHandler,
    Media::{
        MediaPlaybackStatus, MediaPlaybackType, SystemMediaTransportControls,
        SystemMediaTransportControlsButton as Button,
        SystemMediaTransportControlsButtonPressedEventArgs,
    },
    Win32::{
        System::WinRT::{
            ISystemMediaTransportControlsInterop, RoGetActivationFactory, RoInitialize,
            RoUninitialize, RO_INIT_MULTITHREADED,
        },
        UI::WindowsAndMessaging::{
            CreateWindowExW, DestroyWindow, DispatchMessageW, PeekMessageW, TranslateMessage, MSG,
            PM_REMOVE, WINDOW_EX_STYLE, WS_OVERLAPPED,
        },
    },
};

#[derive(Clone, Copy)]
pub enum Action {
    Play,
    Pause,
    Next,
    Previous,
}
#[derive(Clone, PartialEq)]
pub struct Display {
    pub title: String,
    pub artist: String,
    pub playing: bool,
    pub enabled: bool,
}
pub struct Bridge {
    sender: mpsc::SyncSender<Option<Display>>,
}
impl Bridge {
    pub fn start(actions: tokio::sync::mpsc::UnboundedSender<Action>) -> anyhow::Result<Self> {
        let (sender, updates) = mpsc::sync_channel::<Option<Display>>(1);
        let (ready_tx, ready_rx) = mpsc::sync_channel(1);
        std::thread::spawn(move || {
            let result = (|| -> windows::core::Result<()> {
                unsafe {
                    RoInitialize(RO_INIT_MULTITHREADED)?;
                    let window = CreateWindowExW(
                        WINDOW_EX_STYLE::default(),
                        w!("STATIC"),
                        w!("Pomme TUI"),
                        WS_OVERLAPPED,
                        0,
                        0,
                        0,
                        0,
                        None,
                        None,
                        None,
                        None,
                    )?;
                    let factory: ISystemMediaTransportControlsInterop = RoGetActivationFactory(
                        &HSTRING::from("Windows.Media.SystemMediaTransportControls"),
                    )?;
                    let controls: SystemMediaTransportControls = factory.GetForWindow(window)?;
                    controls.SetIsPlayEnabled(true)?;
                    controls.SetIsPauseEnabled(true)?;
                    controls.SetIsNextEnabled(true)?;
                    controls.SetIsPreviousEnabled(true)?;
                    let handler = TypedEventHandler::<
                        SystemMediaTransportControls,
                        SystemMediaTransportControlsButtonPressedEventArgs,
                    >::new(move |_, args| {
                        if let Some(args) = args.as_ref() {
                            let action = match args.Button()? {
                                Button::Play => Some(Action::Play),
                                Button::Pause => Some(Action::Pause),
                                Button::Next => Some(Action::Next),
                                Button::Previous => Some(Action::Previous),
                                _ => None,
                            };
                            if let Some(action) = action {
                                let _ = actions.send(action);
                            }
                        }
                        Ok(())
                    });
                    let token = controls.ButtonPressed(&handler)?;
                    let updater = controls.DisplayUpdater()?;
                    updater.SetType(MediaPlaybackType::Music)?;
                    controls.SetIsEnabled(false)?;
                    let _ = ready_tx.send(Ok(()));
                    let mut previous = None;
                    loop {
                        let mut message = MSG::default();
                        while PeekMessageW(&mut message, None, 0, 0, PM_REMOVE).as_bool() {
                            let _ = TranslateMessage(&message);
                            DispatchMessageW(&message);
                        }
                        match updates.recv_timeout(Duration::from_millis(100)) {
                            Ok(Some(display)) => {
                                if previous.as_ref() == Some(&display) {
                                    continue;
                                }
                                controls.SetIsEnabled(display.enabled)?;
                                controls.SetPlaybackStatus(if display.playing {
                                    MediaPlaybackStatus::Playing
                                } else {
                                    MediaPlaybackStatus::Paused
                                })?;
                                let properties = updater.MusicProperties()?;
                                properties.SetTitle(&HSTRING::from(&display.title))?;
                                properties.SetArtist(&HSTRING::from(&display.artist))?;
                                updater.Update()?;
                                previous = Some(display);
                            }
                            Ok(None) | Err(mpsc::RecvTimeoutError::Disconnected) => break,
                            Err(mpsc::RecvTimeoutError::Timeout) => {}
                        }
                    }
                    let _ = controls.SetIsEnabled(false);
                    let _ = controls.RemoveButtonPressed(token);
                    let _ = DestroyWindow(window);
                    RoUninitialize();
                    Ok(())
                }
            })();
            if result.is_err() {
                let _ = ready_tx.send(Err("Windows media controls could not initialize"));
            }
        });
        ready_rx
            .recv_timeout(Duration::from_secs(5))
            .map_err(|_| anyhow::anyhow!("Windows media controls timed out"))?
            .map_err(anyhow::Error::msg)?;
        Ok(Self { sender })
    }
    pub fn update(&self, display: Display) {
        let _ = self.sender.try_send(Some(display));
    }
}
impl Drop for Bridge {
    fn drop(&mut self) {
        let _ = self.sender.try_send(None);
    }
}
