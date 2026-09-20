package update

// DefaultPublicKey is the base64 ed25519 public key that release manifests are verified against when
// UPDATE_PUBLIC_KEY is not set. Its private half lives only in the RELEASE_SIGNING_KEY CI secret (create the pair
// with `releasetool keygen`). It is a variable so a build can stamp another key with -ldflags and tests can clear it.
// With no key at all the UI update feature reports itself unavailable instead of trusting anything.
var DefaultPublicKey = "YwFcnNQon4chu/ozkNfwcc1LaG1Rvm3+P1DjRlii+I8="
