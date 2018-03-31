package interfaces


type HeightCache interface {
	GetHeight()(height uint64)
	Stop()
}
