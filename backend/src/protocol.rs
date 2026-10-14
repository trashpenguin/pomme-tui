use serde::{Deserialize, Serialize};
use serde_json::Value;

#[derive(Clone, Debug, Default, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct WireTrack {
    pub id: String,
    pub title: String,
    pub artist: String,
    pub album: String,
    pub kind: String,
    pub dur_ms: u64,
    pub art: String,
}

#[derive(Debug, Deserialize)]
pub struct Request {
    pub id: u64,
    pub method: String,
    pub params: Value,
}
pub const MAX_LINE: usize = 1_048_576;
/// Validates the finite method set before dispatch; input contents never appear in errors.
pub fn decode(bytes: &[u8]) -> anyhow::Result<Request> {
    anyhow::ensure!(bytes.len() <= MAX_LINE, "request exceeds size limit");
    let request: Request =
        serde_json::from_slice(bytes).map_err(|_| anyhow::anyhow!("invalid JSON request"))?;
    anyhow::ensure!(request.params.is_object(), "params must be an object");
    anyhow::ensure!(
        [
            "hello",
            "state",
            "spectrum",
            "search",
            "library",
            "play",
            "queue_next",
            "queue_later",
            "play_pause",
            "next",
            "prev",
            "jump",
            "seek",
            "set_volume",
            "toggle_shuffle",
            "cycle_repeat",
            "account",
            "sign_out",
            "reload",
            "shutdown"
        ]
        .contains(&request.method.as_str()),
        "unknown method"
    );
    Ok(request)
}

#[derive(Clone, Debug, Default, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct WireState {
    pub authed: bool,
    pub playing: bool,
    pub initializing: bool,
    pub pos_ms: u64,
    pub dur_ms: u64,
    pub volume: u8,
    pub shuffle: bool,
    pub repeat: u8,
    pub now: Option<WireTrack>,
    pub queue: Vec<WireTrack>,
    pub queue_pos: usize,
    pub err: String,
}

#[derive(Default, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchResults {
    pub songs: Vec<WireTrack>,
    pub albums: Vec<WireTrack>,
    pub playlists: Vec<WireTrack>,
    pub recent: Vec<WireTrack>,
    pub err: String,
}

/// Responses carry one result or a safe error and always retain the caller's ID.
#[derive(Serialize)]
pub struct Response {
    pub id: u64,
    pub result: Option<Value>,
    pub error: Option<String>,
}
impl Response {
    pub fn ok(id: u64, result: impl Serialize) -> Self {
        match serde_json::to_value(result) {
            Ok(value) => Self {
                id,
                result: Some(value),
                error: None,
            },
            Err(_) => Self::error(id, "cannot serialize response"),
        }
    }
    pub fn error(id: u64, error: impl Into<String>) -> Self {
        Self {
            id,
            result: None,
            error: Some(error.into()),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn protocol_accepts_known_request_and_preserves_id() {
        let r = decode(br#"{"id":7,"method":"state","params":{}}"#).unwrap();
        assert_eq!(r.id, 7);
        assert_eq!(r.method, "state");
    }
    #[test]
    fn protocol_rejects_malformed_oversized_and_unknown_requests() {
        for input in [
            b"bad".as_slice(),
            br#"{"id":1,"method":"state","params":[]}"#,
            br#"{"id":1,"method":"secrets","params":{}}"#,
            br#"{"id":-1,"method":"state","params":{}}"#,
        ] {
            assert!(decode(input).is_err());
        }
        assert!(decode(&vec![b' '; 1_048_577]).is_err());
    }
    #[test]
    fn track_wire_uses_milliseconds_and_exposes_no_credentials() {
        let value = serde_json::to_value(WireTrack {
            id: "a".into(),
            dur_ms: 180000,
            ..Default::default()
        })
        .unwrap();
        assert_eq!(value["durMs"], 180000);
        assert!(value.get("token").is_none());
    }
}
