use anyhow::{Context, Result, bail};
use apple_native_engine::auth_failure::{AuthExpired, classify, is_auth_error};
use apple_native_engine::protocol::{DiscoveryFeed, DiscoverySection, SearchResults, WireTrack};
use apple_native_engine::transport::artwork_template;
use music::{Album, Artist, ArtistRef, GenreItem, MusicApi, Playlist, Track};
use std::collections::{HashMap, HashSet};
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
                                art: artwork_template(item.cover.unwrap_or_default()),
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
        if category == "artists" {
            let missing: Vec<String> = result
                .artists
                .iter()
                .filter(|artist| artist.art.is_empty())
                .map(|artist| artist.id.clone())
                .collect();
            if !missing.is_empty() {
                if let Ok(pictures) = self.api.artist_images(missing).await {
                    fill_artist_pictures(&mut result.artists, pictures);
                }
            }
        }
        Ok(result)
    }
    pub async fn artist_page(&self, id: &str) -> Result<SearchResults> {
        let artist = self.api.artist(id).await?;
        Ok(artist_page_results(id, artist))
    }
    pub async fn song_album(&self, id: &str) -> Result<SearchResults> {
        anyhow::ensure!(
            !id.is_empty() && id.len() <= 256 && !id.chars().any(char::is_control),
            "invalid song ID"
        );
        let track = self.api.track(id).await.map_err(classify)?;
        let id = song_album_id(track.album_id)?;
        let detail = self.api.album(&id).await.map_err(classify)?;
        Ok(SearchResults {
            albums: vec![album(detail.album)],
            ..Default::default()
        })
    }
    pub async fn song_artists(&self, id: &str) -> Result<SearchResults> {
        anyhow::ensure!(
            !id.is_empty() && id.len() <= 256 && !id.chars().any(char::is_control),
            "invalid song ID"
        );
        let track = self.api.track(id).await.map_err(classify)?;
        let ids = track
            .artist_refs
            .iter()
            .filter_map(|artist| artist.id.clone())
            .collect();
        // Pictures are decorative. A failed portrait lookup must not discard valid IDs.
        let pictures = self.api.artist_images(ids).await.unwrap_or_default();
        let artists = artist_references(track.artist_refs, pictures);
        anyhow::ensure!(
            !artists.is_empty(),
            "artist information is unavailable for this song"
        );
        Ok(SearchResults {
            artists,
            ..Default::default()
        })
    }
    /// Independent groups can succeed even if Apple declines another result type.
    pub async fn search(&self, query: &str) -> Result<SearchResults> {
        anyhow::ensure!(
            !query.trim().is_empty() && query.len() <= 512,
            "search query must be 1–512 bytes"
        );
        let (songs, albums, playlists, artists) = tokio::join!(
            self.api.search(query),
            self.api.search_albums(query),
            self.api.search_playlists(query),
            self.api.search_artists(query)
        );
        let mut result = SearchResults::default();
        for error in [
            songs.as_ref().err(),
            albums.as_ref().err(),
            playlists.as_ref().err(),
            artists.as_ref().err(),
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
        match artists {
            Ok(items) => {
                result.artists = items
                    .into_iter()
                    .take(50)
                    .filter_map(artist_search_hit)
                    .collect()
            }
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

fn artist_page_results(id: &str, artist: Artist) -> SearchResults {
    SearchResults {
        artists: vec![WireTrack {
            id: id.to_owned(),
            title: artist.name,
            kind: "artist".into(),
            art: artwork_template(artist.cover_large.unwrap_or_default()),
            ..Default::default()
        }],
        songs: artist.top_tracks.into_iter().map(song).collect(),
        albums: artist.albums.into_iter().map(album).collect(),
        ..Default::default()
    }
}

fn fill_artist_pictures(artists: &mut [WireTrack], pictures: HashMap<String, String>) {
    for artist in artists {
        if artist.art.is_empty() {
            if let Some(picture) = pictures.get(&artist.id) {
                artist.art = artwork_template(picture.clone());
            }
        }
    }
}

fn artist_references(
    references: Vec<ArtistRef>,
    pictures: HashMap<String, String>,
) -> Vec<WireTrack> {
    let mut seen = HashSet::new();
    references
        .into_iter()
        .filter_map(|artist| {
            let id = artist.id.filter(|id| !id.is_empty())?;
            if !seen.insert(id.clone()) {
                return None;
            }
            Some(WireTrack {
                art: artwork_template(pictures.get(&id).cloned().unwrap_or_default()),
                id,
                title: artist.name,
                kind: "artist".into(),
                ..Default::default()
            })
        })
        .collect()
}

fn artist_search_hit(artist: music::SavedArtist) -> Option<WireTrack> {
    if artist.id.is_empty() || artist.id.chars().any(char::is_control) {
        return None;
    }
    Some(WireTrack {
        id: artist.id,
        title: artist.name,
        kind: "artist".into(),
        art: artwork_template(artist.cover.unwrap_or_default()),
        ..Default::default()
    })
}

fn song_album_id(id: Option<String>) -> Result<String> {
    let id = id.filter(|id| !id.is_empty() && id.len() <= 256 && !id.chars().any(char::is_control));
    id.ok_or_else(|| anyhow::anyhow!("album information is unavailable for this song"))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn artist_search_preserves_catalog_ids_and_tolerates_missing_portraits() {
        let item = artist_search_hit(music::SavedArtist {
            id: "artist-7".into(),
            name: "Artist".into(),
            cover: None,
            added_at: None,
        })
        .unwrap();
        assert_eq!(item.id, "artist-7");
        assert_eq!(item.kind, "artist");
        assert!(item.art.is_empty());
        assert!(
            artist_search_hit(music::SavedArtist {
                id: "".into(),
                name: "Invalid".into(),
                cover: None,
                added_at: None
            })
            .is_none()
        );
    }

    #[test]
    fn song_album_reference_requires_a_real_valid_id() {
        assert_eq!(
            song_album_id(Some("catalog-album-7".into())).unwrap(),
            "catalog-album-7"
        );
        for id in [
            None,
            Some("".into()),
            Some("bad\n".into()),
            Some("x".repeat(257)),
        ] {
            assert!(song_album_id(id).is_err());
        }
    }

    #[test]
    fn missing_library_portraits_are_filled_by_id_without_replacing_existing_art() {
        let mut artists = vec![
            WireTrack {
                id: "one".into(),
                art: "existing".into(),
                ..Default::default()
            },
            WireTrack {
                id: "two".into(),
                ..Default::default()
            },
        ];
        fill_artist_pictures(
            &mut artists,
            HashMap::from([
                ("one".into(), "replacement".into()),
                ("two".into(), "https://example.test/photo.jpg".into()),
            ]),
        );
        assert_eq!(artists[0].art, "existing");
        assert!(!artists[1].art.is_empty());
    }
    #[test]
    fn song_references_preserve_real_ids_without_pictures_and_deduplicate() {
        let artists = artist_references(
            vec![
                ArtistRef {
                    name: "Artist".into(),
                    id: Some("catalog-7".into()),
                },
                ArtistRef {
                    name: "Again".into(),
                    id: Some("catalog-7".into()),
                },
                ArtistRef {
                    name: "Unknown".into(),
                    id: None,
                },
            ],
            HashMap::new(),
        );
        assert_eq!(artists.len(), 1);
        assert_eq!(artists[0].id, "catalog-7");
        assert_eq!(artists[0].title, "Artist");
        assert_eq!(artists[0].kind, "artist");
        assert!(artists[0].art.is_empty());
    }
    #[test]
    fn artist_page_includes_portrait_profile_without_losing_the_catalogue() {
        let result = artist_page_results(
            "artist-1",
            Artist {
                name: "Artist".into(),
                cover_large: Some("https://example.test/portrait/{w}x{h}.jpg".into()),
                biography: None,
                monthly_listeners: None,
                top_tracks: Vec::new(),
                albums: Vec::new(),
            },
        );
        assert_eq!(result.artists[0].id, "artist-1");
        assert_eq!(result.artists[0].title, "Artist");
        assert!(!result.artists[0].art.is_empty());
        assert!(result.songs.is_empty() && result.albums.is_empty());
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
