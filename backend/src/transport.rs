use crate::protocol::WireTrack;
/// Restores Apple's size placeholders so terminal tiles request small images.
pub fn artwork_template(url: String) -> String {
    let Some((base, file)) = url.rsplit_once('/') else {
        return url;
    };
    let Some(x) = file.find('x') else {
        return url;
    };
    if x == 0 || !file[..x].bytes().all(|b| b.is_ascii_digit()) {
        return url;
    }
    let height = file[x + 1..]
        .bytes()
        .take_while(|b| b.is_ascii_digit())
        .count();
    if height == 0 {
        return url;
    }
    format!("{base}/{{w}}x{{h}}{}", &file[x + 1 + height..])
}
/// Rejects empty/unplayable resolutions and work superseded by a newer selection.
pub fn validate_resolution(
    generation: u64,
    current: u64,
    tracks: &[WireTrack],
) -> anyhow::Result<()> {
    anyhow::ensure!(generation == current, "selection superseded");
    anyhow::ensure!(!tracks.is_empty(), "item has no playable catalog tracks");
    anyhow::ensure!(
        tracks.iter().all(|t| !t.id.is_empty() && t.kind == "song"),
        "item has no playable catalog tracks"
    );
    Ok(())
}
pub fn clamp_volume(volume: i64) -> u8 {
    volume.clamp(0, 100) as u8
}
pub fn clamp_seek(position: u64, duration: u64) -> u64 {
    if duration == 0 {
        position
    } else {
        position.min(duration.saturating_sub(250))
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn native_artwork_sizes_can_be_requested_by_the_frontend() {
        assert_eq!(
            artwork_template("https://example.test/image/600x600bb.jpg".into()),
            "https://example.test/image/{w}x{h}bb.jpg"
        );
        assert_eq!(
            artwork_template("https://example.test/image/{w}x{h}bb.jpg".into()),
            "https://example.test/image/{w}x{h}bb.jpg"
        );
    }
    #[test]
    fn failed_or_stale_resolution_cannot_replace_queue() {
        assert!(validate_resolution(1, 1, &[]).is_err());
        let track = WireTrack {
            id: "a".into(),
            kind: "song".into(),
            ..Default::default()
        };
        assert!(validate_resolution(1, 2, &[track.clone()]).is_err());
        assert!(validate_resolution(2, 2, &[track]).is_ok());
    }
    #[test]
    fn volume_and_seek_have_safe_bounds() {
        assert_eq!(clamp_volume(-100), 0);
        assert_eq!(clamp_volume(200), 100);
        assert_eq!(clamp_seek(100000, 20000), 19750);
        assert_eq!(clamp_seek(5000, 20000), 5000);
    }
}
