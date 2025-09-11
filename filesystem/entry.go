package filesystem

type Entry struct {
	IsDirectory bool
	Children    []*Entry
	Parent      *Entry
	Size        uint64
	Name        string
	Path        string
	Err         error
}

func newEntry() *Entry {
	return &Entry{
		IsDirectory: false,
		Children:    []*Entry{},
		Parent:      nil,
		Size:        0,
		Name:        "",
		Path:        "",
		Err:         nil,
	}
}
