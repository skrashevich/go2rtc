package plugin

type Plugin interface {
	Start()
	RegisterHandlers()
}
