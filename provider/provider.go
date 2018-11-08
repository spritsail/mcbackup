package provider

type Provider interface {
	TakeBackup() error
}
