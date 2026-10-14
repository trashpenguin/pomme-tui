use std::{
    io::Read,
    sync::{Arc, atomic::AtomicBool},
};
/// EOF on the frontend-owned pipe also cancels sign-in if the frontend crashes or is closed.
pub fn monitor_owner(mut reader: impl Read + Send + 'static) -> Arc<AtomicBool> {
    let canceled = Arc::new(AtomicBool::new(false));
    let signal = canceled.clone();
    std::thread::spawn(move || {
        let mut byte = [0; 1];
        loop {
            match reader.read(&mut byte) {
                Ok(0) => break,
                Ok(_) => {}
                Err(error) if error.kind() == std::io::ErrorKind::Interrupted => continue,
                Err(_) => break,
            }
        }
        signal.store(true, std::sync::atomic::Ordering::Release);
    });
    canceled
}
#[cfg(test)]
mod tests {
    use super::*;
    use std::{
        sync::atomic::Ordering,
        time::{Duration, Instant},
    };
    #[test]
    fn owner_eof_cancels_signin_without_a_frontend_callback() {
        let canceled = monitor_owner(std::io::Cursor::new(Vec::<u8>::new()));
        let until = Instant::now() + Duration::from_secs(1);
        while !canceled.load(Ordering::Acquire) && Instant::now() < until {
            std::thread::sleep(Duration::from_millis(5));
        }
        assert!(
            canceled.load(Ordering::Acquire),
            "owned login survived parent EOF"
        );
    }
}
