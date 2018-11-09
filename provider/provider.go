package provider

type Provider interface {
	TakeBackup(name string) error
}
