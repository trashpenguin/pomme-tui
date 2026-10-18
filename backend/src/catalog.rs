use anyhow::{Context, Result, bail};
use apple_native_engine::auth_failure::{AuthExpired, classify, is_auth_error};
use apple_native_engine::protocol::{DiscoveryFeed, DiscoverySection, SearchResults, WireTrack};
use apple_native_engine::transport::artwork_template;
use music::{Album, GenreItem, MusicApi, Playlist, Track};
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
    pub async fn discovery(&self) -> Result<DiscoveryFeed> {
        let feed = self.api.home().await?;
        let sections = feed
            .sections
            .into_iter()
            .filter_map(|section| {
                let items: Vec<WireTrack> = section
                    .items
                    .into_iter()
                    .filter_map(discovery_item)
                    .collect();
                (!items.is_empty()).then_some(DiscoverySection {
                    title: section.title,
                    items,
                })
            })
            .collect();
        Ok(DiscoveryFeed { sections })
    }

    pub async fn discovery_tracks(&self, kind: &str, id: &str) -> Result<Vec<WireTrack>> {
        if kind == "genre" {
            anyhow::ensure!(
                !id.is_empty() && id.len() <= 256 && !id.chars().any(char::is_control),
                "invalid catalog item ID"
            );
            let detail = self
                .api
                .genre(id)
                .await
                .map_err(classify)
                .context("genre lookup failed")?;
            let mut items = Vec::new();
            for section in detail.sections {
                for item in section.items {
                    if let Some(item) = discovery_item(item) {
                        if matches!(item.kind.as_str(), "song" | "album" | "playlist") {
                            items.push(item);
                            if items.len() == 1000 {
                                return Ok(items);
                            }
                        }
                    }
                }
            }
            anyhow::ensure!(!items.is_empty(), "genre has no playable recommendations");
            return Ok(items);
        }
        self.resolve(kind, id).await
    }

    pub async fn save_to_library(&self, kind: &str, id: &str) -> Result<()> {
        anyhow::ensure!(
            !id.is_empty() && id.len() <= 256 && !id.chars().any(char::is_control),
            "invalid catalog item ID"
        );
        match kind {
            "song" => {
                self.api
                    .set_in_library(music::MediaKind::Track, id, true)
                    .await
            }
            "album" => {
                self.api
                    .set_in_library(music::MediaKind::Album, id, true)
                    .await
            }
            "playlist" => self.api.add_playlist_to_library(id).await,
            _ => bail!("this item cannot be added to the library"),
        }
    }

    pub async fn save_queue(&self, name: &str, tracks: &[WireTrack]) -> Result<SearchResults> {
        anyhow::ensure!(
            !name.trim().is_empty() && name.len() <= 120 && !tracks.is_empty(),
            "invalid playlist"
        );
        let id = self.api.create_playlist(name).await?;
        for (index, track) in tracks.iter().enumerate() {
            if self
                .api
                .add_track_to_playlist(&id, &track.id)
                .await
                .is_err()
            {
                return Ok(SearchResults {
                    err: format!(
                        "Playlist created, but only {} of {} tracks were added",
                        index,
                        tracks.len()
                    ),
                    ..Default::default()
                });
            }
        }
        Ok(SearchResults::default())
    }
    pub async fn library_page(&self, category: &str, offset: usize) -> Result<SearchResults> {
        anyhow::ensure!(offset <= 10000, "library page is too far");
        let mut result = SearchResults::default();
        match category {
            "songs" | "favorites" => {
                let mut pages = if category == "favorites" {
                    self.api.saved_tracks_paged().await?
                } else {
                    self.api.all_tracks_paged().await?
                };
                let mut seen = 0;
                while let Some(page) = pages.recv().await {
                    for track in page?.items {
                        if seen >= offset && result.songs.len() < 50 {
                            result.songs.push(song(track))
                        }
                        seen += 1;
                    }
                    if result.songs.len() == 50 {
                        break;
                    }
                }
            }
            "albums" => {
                let mut pages = self.api.all_albums_paged().await?;
                let mut seen = 0;
                while let Some(page) = pages.recv().await {
                    for item in page?.items {
                        if seen >= offset && result.albums.len() < 50 {
                            result.albums.push(album(item))
                        }
                        seen += 1;
                    }
                    if result.albums.len() == 50 {
                        break;
                    }
                }
            }
            "artists" => {
                let mut pages = self.api.all_artists_paged().await?;
                let mut seen = 0;
                while let Some(page) = pages.recv().await {
                    for item in page?.items {
                        if seen >= offset && result.artists.len() < 50 {
                            result.artists.push(WireTrack {
                                id: item.id,
                                title: item.name,
                                kind: "artist".into(),
                                art: item.cover.unwrap_or_default(),
                                ..Default::default()
                            })
                        }
                        seen += 1;
                    }
                    if result.artists.len() == 50 {
                        break;
                    }
                }
            }
            "playlists" => {
                result.playlists = self
                    .api
                    .playlists()
                    .await?
                    .into_iter()
                    .skip(offset)
                    .take(50)
                    .map(playlist)
                    .collect()
            }
            _ => bail!("invalid library category"),
        }
        Ok(result)
    }
    pub async fn artist_page(&self, id: &str) -> Result<SearchResults> {
        let artist = self.api.artist(id).await?;
        Ok(SearchResults {
            songs: artist.top_tracks.into_iter().map(song).collect(),
            albums: artist.albums.into_iter().map(album).collect(),
            ..Default::default()
        })
    }
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

fn discovery_item(item: GenreItem) -> Option<WireTrack> {
    match item {
        GenreItem::Playlist(item) => Some(WireTrack {
            id: item.id,
            title: item.name,
            artist: item.owner,
            kind: "playlist".into(),
            art: artwork_template(item.cover.unwrap_or_default()),
            ..Default::default()
        }),
        GenreItem::Album(item) => Some(WireTrack {
            id: item.id,
            title: item.name,
            artist: item.artists,
            kind: "album".into(),
            art: artwork_template(item.cover_large.or(item.cover).unwrap_or_default()),
            ..Default::default()
        }),
        GenreItem::Track(item) => Some(song(item)),
        GenreItem::Artist(item) => Some(WireTrack {
            id: item.id,
            title: item.name,
            kind: "artist".into(),
            art: artwork_template(item.cover.unwrap_or_default()),
            ..Default::default()
        }),
        GenreItem::Genre(item) => Some(WireTrack {
            id: item.id,
            title: item.name,
            kind: "genre".into(),
            art: artwork_template(item.cover.unwrap_or_default()),
            ..Default::default()
        }),
    }
}
