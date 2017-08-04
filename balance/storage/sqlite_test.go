package storage

import (
	"testing"
)



// TODO: Move all test to STORAGE_MEMORY_TEST
func TestBase(t *testing.T) {

	store, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Error(err)
	}

	height, err1 := store.GetHeight()
	if height != -1 || err1 != nil {
		t.Error("GetHeight error", height, err1)
	}

	store.SetHeight(44)
	
	height, err1 = store.GetHeight()
	if height != 44 || err1 != nil {
		t.Error("GetHeight error", height, err1)
	}

	store.SetHeight(4567)
	
	height, err1 = store.GetHeight()
	if height != 4567 || err1 != nil {
		t.Error("GetHeight error", height, err1)
	}





	if length, err := store.Len(); length != 0 || err != nil {
		t.Error("A new database should be empty")
	}

	store.Set("asdf", 12)
	
	if length, err := store.Len(); length != 1 || err != nil {
		t.Error("A new database should be empty")
	}

	balance, err := store.Get("asdf")
	if err != nil {
		t.Error(err)
	}
	if balance != 12 {
		t.Errorf("Unexpected value %v", balance)
	}

	contains, err := store.Contains("asdf")
	if !contains {
		t.Error("It should contain asdf")
	}
	
	contains, err = store.Contains("dsfhgasfdasdfasdf")
	if contains {
		t.Error("It shouldn't contain asdf")
	}


	balance, err = store.Get("asdfasdf")
	if err != nil {
		t.Error(err)
	}
	if balance != 0 {
		t.Errorf("Unexpected value %v", balance)
	}

	err = store.Delete("11111111")
	if err != nil {
		t.Error(err)
	}	
	 
	if length, err := store.Len(); length != 1 || err != nil {
		t.Error("It should be one balance in the db")
	}

	err = store.Delete("asdf")
	if err != nil {
		t.Error(err)
	}	
	 
	if length, err := store.Len(); length != 0 || err != nil {
		t.Error("A new database should be empty")
	}


}
