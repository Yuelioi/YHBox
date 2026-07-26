//go:build !windows

package securestore

type unavailableStore struct{}

func New() Store { return unavailableStore{} }

func (unavailableStore) Get(string) (string, error) { return "", ErrUnavailable }
func (unavailableStore) Set(string, string) error   { return ErrUnavailable }
func (unavailableStore) Delete(string) error        { return ErrUnavailable }
