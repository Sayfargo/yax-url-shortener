package cache

import (
	"fmt"
)

// Example демонстрирует базовый цикл работы с кэшем:
// инициализацию, сохранение значения и его последующее извлечение
// с приведением типа (type assertion).
func Example() {
	// Инициализируем in-memory кэш
	c := Init()

	// Сохраняем значение в кэш
	c.Set("key", 64)

	// Извлекаем сохраненное значение
	v, err := c.Get("key")
	if err != nil {
		fmt.Printf("failed to get value: %v\n", err)
		return
	}

	// Выполняем приведение типа из any в int
	num, ok := v.(int)
	fmt.Printf("Got: value=%d, ok=%t", num, ok)

	// Output:
	// Got: value=64, ok=true
}

// ExampleCache_Snapshot демонстрирует изоляцию данных при создании снимка (snapshot).
// Изменения, внесенные в кэш после вызова Snapshot, не попадают в созданную копию.
func ExampleCache_Snapshot() {
	c := Init()

	// Наполняем кэш начальными данными
	c.Set("1", 1)
	c.Set("2", 2)
	c.Set("3", 3)

	// Создаем снимок текущего состояния кэша
	s := c.Snapshot()

	// Добавляем новое значение в кэш после создания снимка
	c.Set("4", 4)

	// Проверяем, что в снимке отсутствует значение, добавленное позже
	_, ok := s["4"]
	fmt.Printf("Got: ok=%t", ok)

	// Output:
	// Got: ok=false
}
