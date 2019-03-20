package controller

// Handler ...
type Handler interface {
	// ObjectCreated handles object creation
	ObjectCreated(obj interface{})
	// ObjectUpdated handles object update
	ObjectUpdated(new interface{})
	// ObjectDeleted handles object deletion
	ObjectDeleted(obj interface{})
}
