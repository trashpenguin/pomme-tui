use anyhow::{Context, Result, bail};
use apple_native_engine::auth_failure::{AuthExpired, classify, is_auth_error};
use apple_native_engine::protocol::{SearchResults, WireTrack};
use apple_native_engine::transport::artwork_template;
use music::{Album, MusicApi, Playlist, Track};
use std::sync::Arc;

#[derive(Clone)]
pub struct Catalog {
    pub api: Arc<dyn MusicApi>,
}
pub fn song(track: Track) -> WireTrack {
    WireTrack {
        id: track.id.unwrap_or_default(),
        title: track.name,
        artist: track.artists,
        album: track.album,
        kind: "song".into(),
        dur_ms: track.duration.as_millis() as u64,
        art: artwork_template(track.cover.unwrap_or_default()),
    }
}
fn album(album: Album) -> WireTrack {
    WireTrack {
        id: album.id,
        title: album.name,
        artist: album.artists,
        kind: "album".into(),
        art: artwork_template(album.cover_large.or(album.cover).unwrap_or_default()),
        ..Default::default()
    }
}
fn playlist(playlist: Playlist) -> WireTrack {
    WireTrack {
        id: playlist.id,
        title: playlist.name,
        kind: "playlist".into(),
        art: artwork_template(playlist.cover.unwrap_or_default()),
        ..Default::default()
    }
}
impl Catalog {
    /// Independent groups can succeed even if Apple declines another result type.
    pub async fn search(&self, query: &str) -> Result<SearchResults> {
        anyhow::ensure!(
            !query.trim().is_empty() && query.len() <= 512,
            "search query must be 1–512 bytes"
        );
        let (songs, albums, playlists) = tokio::join!(
            self.api.search(query),
            self.api.search_albums(query),
            self.api.search_playlists(query)
        );
        let mut result = SearchResults::default();
        for error in [
            songs.as_ref().err(),
            albums.as_ref().err(),
            playlists.as_ref().err(),
        ]
        .into_iter()
        .flatten()
        {
            if is_auth_error(error) {
                return Err(AuthExpired.into());
            }
        }
        match songs {
            Ok(items) => result.songs = items.into_iter().take(50).map(song).collect(),
            Err(_) => result.err = "Some catalog results are unavailable".into(),
        }
        match albums {
            Ok(items) => result.albums = items.into_iter().take(50).map(album).collect(),
            Err(_) => result.err = "Some catalog results are unavailable".into(),
        }
        match playlists {
            Ok(items) => result.playlists = items.into_iter().take(50).map(playlist).collect(),
            Err(_) => result.err = "Some catalog results are unavailable".into(),
        }
        Ok(result)
    }
    async fn albums(&self) -> Result<Vec<WireTrack>> {
        let mut pages = self.api.all_albums_paged().await?;
        let mut items = Vec::new();
        while let Some(page) = pages.recv().await {
            items.extend(page?.items.into_iter().map(album));
            if items.len() >= 100 {
                break;
            }
        }
        items.truncate(100);
        Ok(items)
    }
    pub async fn library(&self) -> Result<SearchResults> {
        let (albums, playlists, recent) = tokio::join!(
            self.albums(),
            self.api.playlists(),
            self.api.recently_played()
        );
        let mut result = SearchResults::default();
        for error in [
            albums.as_ref().err(),
            playlists.as_ref().err(),
            recent.as_ref().err(),
        ]
        .into_iter()
        .flatten()
        {
            if is_auth_error(error) {
                return Err(AuthExpired.into());
            }
        }
        match albums {
            Ok(items) => result.albums = items,
            Err(_) => result.err = "Some library items are unavailable".into(),
        }
        match playlists {
            Ok(items) => result.playlists = items.into_iter().take(100).map(playlist).collect(),
            Err(_) => result.err = "Some library items are unavailable".into(),
        }
        match recent {
            Ok(items) => result.recent = items.into_iter().take(10).map(song).collect(),
            Err(_) => result.err = "Some account history is unavailable".into(),
        }
        Ok(result)
    }
    pub async fn resolve(&self, kind: &str, id: &str) -> Result<Vec<WireTrack>> {
        anyhow::ensure!(
            !id.is_empty() && id.len() <= 256 && !id.chars().any(char::is_control),
            "invalid catalog item ID"
        );
        let items = match kind {
            "song" => vec![
                self.api
                    .track(id)
                    .await
                    .map_err(classify)
                    .context("song lookup failed")?,
            ],
            "album" => self
                .api
                .album_tracks(id)
                .await
                .map_err(classify)
                .context("album lookup failed")?,
            "playlist" => self
                .api
                .playlist_tracks(id)
                .await
                .map_err(classify)
                .context("playlist lookup failed")?,
            _ => bail!("unknown item kind"),
        };
        let items: Vec<_> = items
            .into_iter()
            .filter(|t| t.playable && t.id.is_some())
            .take(1000)
            .map(song)
            .collect();
        anyhow::ensure!(!items.is_empty(), "item has no playable catalog tracks");
        Ok(items)
    }
}
