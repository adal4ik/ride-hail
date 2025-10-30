package driver

type IAuthSerive interface {
	ValidateDriverToken(tokenString string) (string, error)
}
