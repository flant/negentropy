package usecase

type Deleter interface {
	Delete(string) error
}

func deleterSequence(deleters ...Deleter) Deleter {
	return &chainDeleter{deleters: deleters}
}

type chainDeleter struct {
	deleters []Deleter
}

func (d *chainDeleter) Delete(id string) error {
	for _, del := range d.deleters {
		err := del.Delete(id)
		if err != nil {
			return err
		}
	}
	return nil
}
