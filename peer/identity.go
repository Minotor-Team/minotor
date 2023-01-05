package peer

type Identity interface {
	InitIdentityCheck(name, email, phone string)
}
