use anyhow::{Context, Result, bail};
use music::MusicProvider;
use std::time::Duration;
#[cfg(windows)]
pub fn login_cookie(
    provider: &impl MusicProvider,
    canceled: &std::sync::atomic::AtomicBool,
) -> Result<String> {
    use windows::Win32::System::Com::{COINIT_APARTMENTTHREADED, CoInitializeEx, CoUninitialize};
    use windows::Win32::UI::WindowsAndMessaging::{
        DispatchMessageW, MSG, PM_REMOVE, PeekMessageW, TranslateMessage,
    };
    unsafe { CoInitializeEx(None, COINIT_APARTMENTTHREADED) }
        .ok()
        .context("Cannot initialize Windows sign-in thread")?;
    struct Apartment;
    impl Drop for Apartment {
        fn drop(&mut self) {
            unsafe { CoUninitialize() };
        }
    }
    let _apartment = Apartment;
    let sign_in = provider
        .web_sign_in()
        .context("No browser sign-in available")?;
    let mut page = webview::Page::open(webview::Target {
        url: sign_in.url.to_owned(),
        landing: sign_in.landing.to_owned(),
        domain: sign_in.domain.to_owned(),
        proof: sign_in.proof.iter().map(|s| s.to_string()).collect(),
        title: "Apple Music - Sign in".to_owned(),
        agent: sign_in.agent.map(str::to_owned),
        script: None,
    })?;
    let deadline = std::time::Instant::now() + Duration::from_secs(600);
    loop {
        anyhow::ensure!(
            !canceled.load(std::sync::atomic::Ordering::Acquire),
            "Sign-in owner closed"
        );
        let mut msg = MSG::default();
        unsafe {
            while PeekMessageW(&mut msg, None, 0, 0, PM_REMOVE).as_bool() {
                let _ = TranslateMessage(&msg);
                DispatchMessageW(&msg);
            }
        }
        match page.poll() {
            webview::Poll::Cookies(header) => return Ok(header),
            webview::Poll::Closed => bail!("Sign-in cancelled"),
            webview::Poll::Pending => {}
        }
        if std::time::Instant::now() >= deadline {
            bail!("Sign-in timed out");
        }
        std::thread::sleep(Duration::from_millis(50));
    }
}
