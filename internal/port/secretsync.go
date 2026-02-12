package port

type SecretSyncer interface {
	Sync(repo string, secrets map[string]string) error
}
