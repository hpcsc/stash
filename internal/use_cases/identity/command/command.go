package command

type SignIn struct {
	Subject  string
	Provider string
}

type SignOut struct {
	Subject string
}
