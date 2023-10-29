package idgen

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddRandomFilter(t *testing.T) {
	assert := assert.New(t)
	// test overflow
	for i := 0; i <= 64-8; i++ {
		f := AddRandomFilter(uint(i))
		rawId := rand.Intn(0xff) // 8bit
		id, err := f(int64(rawId))
		if err != nil {
			t.Errorf("random filter ret err: %s ", err.Error())
		}

		assert.NotEqual(id^(0xff<<i), 1, "random part is empty. id:%d, i:%d", id, i)
		assert.Equal(id>>i, int64(rawId), "id part err. raw_id:%d, id:%d, i:%d", rawId, id, i)
	}

	// test overflow
	for i := 64 - 8; i <= 64; i++ {
		f := AddRandomFilter(uint(i))
		rawId := 0xff // 8bit
		_, err := f(int64(rawId))

		assert.NotNil(err, "random filter overflow check err. i:%d", i)
	}
}
