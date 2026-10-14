/// Classifies only authentication failures, not network, rate-limit, or unavailable-item errors.
pub fn is_auth_error(error: &anyhow::Error) -> bool {
    error.chain().any(|cause| {
        if cause.downcast_ref::<AuthExpired>().is_some() {
            return true;
        }
        let message = cause.to_string().to_ascii_lowercase();
        [
            "401 unauthorized",
            "403 forbidden",
            "http 401",
            "http 403",
            "authentication expired",
        ]
        .iter()
        .any(|pattern| message.contains(pattern))
    })
}
#[derive(Debug)]
pub struct AuthExpired;
impl std::fmt::Display for AuthExpired {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("Apple Music authentication expired; sign in again")
    }
}
impl std::error::Error for AuthExpired {}
pub fn classify(error: anyhow::Error) -> anyhow::Error {
    if is_auth_error(&error) {
        AuthExpired.into()
    } else {
        error
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn authentication_failures_survive_provider_context_without_matching_other_failures() {
        assert!(is_auth_error(
            &anyhow::anyhow!(
                "apple music answered 401 Unauthorized for GET /me/library: token expired"
            )
            .context("library failed")
        ));
        assert!(is_auth_error(&anyhow::anyhow!(
            "apple refused the license: http 403 Forbidden"
        )));
        assert!(!is_auth_error(&anyhow::anyhow!(
            "apple music answered 404 Not Found"
        )));
        assert!(!is_auth_error(&anyhow::anyhow!("cannot reach apple music")));
        assert!(!is_auth_error(&anyhow::anyhow!(
            "apple music answered 429 Too Many Requests"
        )));
    }
}
