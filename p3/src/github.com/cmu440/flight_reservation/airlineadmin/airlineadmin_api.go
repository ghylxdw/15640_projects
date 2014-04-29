package airlineadmin

const shortForm = "2006.1.2"

type Airlineadmin interface {
	AddFlight(username string, flightId string, number string) error
	CancelFlight(flightId string, date string) error
	EditFlight(flightId string, date string, number string) error

	Close() error
}
