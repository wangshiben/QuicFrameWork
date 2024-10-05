package Session

type sessionError string

func (s sessionError) Error() string {
	return string(s)
}
