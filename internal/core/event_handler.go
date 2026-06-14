package core

type EventHandler[T any] = func(event T)
