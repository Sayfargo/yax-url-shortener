// Модуль generator предоставляет криптографически стойкий генератор случайных
// строк (идентификаторов) фиксированной длины на основе кастомного алфавита.
//
// Реализация использует побитовые маски и расчет шага энтропии для минимизации
// выделения памяти и количества обращений к системному источнику случайных чисел.
package generator

import (
	"crypto/rand"
	"errors"
	"math"
)

// Generator инкапсулирует параметры для быстрой и безопасной генерации
// коротких идентификаторов.
type Generator struct {
	chars []byte
	mask  byte
	step  int
	size  int
}

// New создает и настраивает новый экземпляр [Generator].
//
// Функция рассчитывает оптимальный размер буфера случайных байт (step) на основе
// длины алфавита и целевого размера идентификатора, чтобы свести к минимуму
// накладные расходы на системные вызовы CSPRNG.
//
// Ограничения:
//   - Переданный алфавит (alphabet) не должен быть пустым и не должен превышать 255 символов.
//   - Желаемый размер идентификатора (size) должен быть строго больше нуля.
//
// Возвращает ошибку, если входные параметры не соответствуют ограничениям.
func New(alphabet string, size int) (*Generator, error) {

	if len(alphabet) == 0 || len(alphabet) > 255 {
		return nil, errors.New("alphabet must not be empty and contain no more than 255 chars")
	}
	if size <= 0 {
		return nil, errors.New("size must be positive integer")
	}

	mask := getMask(len(alphabet))

	step := int(math.Ceil(
		1.6 * float64(mask*size) / float64(len(alphabet)),
	))

	return &Generator{
		chars: []byte(alphabet),
		mask:  byte(mask),
		step:  step,
		size:  size,
	}, nil
}

// getMask generates bit mask used to obtain bits from the random bytes that are used to get index of random character
// from the alphabet. Example: if the alphabet has 6 = (110)_2 characters it is sufficient to use mask 7 = (111)_2
func getMask(alphabetSize int) int {
	for i := 1; i <= 8; i++ {
		mask := (2 << uint(i)) - 1
		if mask >= alphabetSize-1 {
			return mask
		}
	}
	return 0
}

// Generate генерирует и возвращает уникальную строку (идентификатор) заданной длины.
//
// Для обеспечения криптографической стойкости чтение случайных байт производится
// через [rand.Read] (системный криптографически стойкий генератор псевдослучайных чисел).
// Метод полностью потокобезопасен, если сам экземпляр [Generator] не модифицируется конкурентно.
func (g *Generator) Generate() (string, error) {

	id := make([]byte, g.size)
	bytes := make([]byte, g.step)
	for j := 0; ; {
		_, err := rand.Read(bytes)
		if err != nil {
			return "", err
		}
		for i := 0; i < g.step; i++ {
			currByte := bytes[i] & byte(g.mask)
			if currByte < byte(len(g.chars)) {
				id[j] = g.chars[currByte]
				j++
				if j == g.size {
					return string(id[:g.size]), nil
				}
			}
		}
	}
}
