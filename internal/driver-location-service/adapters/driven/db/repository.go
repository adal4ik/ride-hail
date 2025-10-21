package db

type Repository struct {
	DriverRepository *DriverRepository
}

func New(db *DataBase) *Repository {
	return &Repository{
		DriverRepository: NewDriverRepository(db),
	}
}
