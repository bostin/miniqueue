package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const tmpDBPath = "/tmp/miniqueue_test_db"

// Insert
func TestInsert_Single(t *testing.T) {
	s := newStore(tmpDBPath)
	t.Cleanup(s.Destroy)

	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value")))

	val, _, err := s.GetNext(defaultTopic)
	assert.NoError(t, err)
	assert.Equal(t, "test_value", string(val))
}

func TestInsert_TwoSameTopic(t *testing.T) {
	s := newStore(tmpDBPath).(*store)
	t.Cleanup(s.Destroy)

	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_1")))
	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_2")))

	val, err := getOffset(s.db, topicFmt, defaultTopic, 0)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_1", string(val))

	val, err = getOffset(s.db, topicFmt, defaultTopic, 1)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_2", string(val))
}

func TestInsert_ThreeSameTopic(t *testing.T) {
	s := newStore(tmpDBPath).(*store)
	t.Cleanup(s.Destroy)

	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_1")))
	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_2")))
	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_3")))

	val, err := getOffset(s.db, topicFmt, defaultTopic, 0)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_1", string(val))

	val, err = getOffset(s.db, topicFmt, defaultTopic, 1)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_2", string(val))

	val, err = getOffset(s.db, topicFmt, defaultTopic, 2)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_3", string(val))
}

// GetNext
func TestGetNext(t *testing.T) {
	s := newStore(tmpDBPath)
	t.Cleanup(s.Destroy)

	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_1")))
	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_2")))
	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_3")))

	val, offset, err := s.GetNext(defaultTopic)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_1", string(val))
	assert.Equal(t, 0, offset)

	val, offset, err = s.GetNext(defaultTopic)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_2", string(val))
	assert.Equal(t, 1, offset)

	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_4")))

	val, offset, err = s.GetNext(defaultTopic)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_3", string(val))
	assert.Equal(t, 2, offset)

	val, offset, err = s.GetNext(defaultTopic)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_4", string(val))
	assert.Equal(t, 3, offset)
}

func TestGetNext_TopicNotInitialised(t *testing.T) {
	s := newStore(tmpDBPath)
	t.Cleanup(s.Destroy)

	val, _, err := s.GetNext(defaultTopic)
	assert.Equal(t, errTopicNotExist, err)
	assert.Equal(t, "", string(val))
}

// Ack
func TestAck(t *testing.T) {
	s := newStore(tmpDBPath).(*store)
	t.Cleanup(s.Destroy)

	ackOffset := 1
	key := []byte(fmt.Sprintf(ackTopicFmt, defaultTopic, ackOffset))
	assert.NoError(t, s.db.Put(key, []byte("hello_world"), nil))

	assert.NoError(t, s.Ack(defaultTopic, ackOffset))

	has, err := s.db.Has(key, nil)
	assert.NoError(t, err)
	assert.False(t, has)
}

func TestAckWithPos(t *testing.T) {
	s := newStore(tmpDBPath).(*store)
	t.Cleanup(s.Destroy)

	err := s.Insert(defaultTopic, []byte("test_value_1"))
	assert.NoError(t, err)

	val, ackOffset, err := s.GetNext(defaultTopic)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_1", string(val))

	val, err = getOffset(s.db, ackTopicFmt, defaultTopic, ackOffset)
	assert.NoError(t, err)
	assert.Equal(t, "test_value_1", string(val))

	assert.NoError(t, s.Ack(defaultTopic, ackOffset))

	_, err = getOffset(s.db, ackTopicFmt, defaultTopic, ackOffset)
	assert.Error(t, err)
}

// Nack
func TestNack(t *testing.T) {
	s := newStore(tmpDBPath)
	t.Cleanup(s.Destroy)

	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_1")))

	_, offset, err := s.GetNext(defaultTopic)
	assert.NoError(t, err)

	assert.NoError(t, s.Nack(defaultTopic, offset))
}

func TestNackTwice(t *testing.T) {
	s := newStore(tmpDBPath)
	t.Cleanup(s.Destroy)

	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_1")))

	_, offset, err := s.GetNext(defaultTopic)
	assert.NoError(t, err)

	// First Nack
	assert.NoError(t, s.Nack(defaultTopic, offset))

	// Second Nack
	err = s.Nack(defaultTopic, offset)
	assert.Error(t, err)
	assert.Equal(t, err, errNackMsgNotExist)
}

func TestNackAndGet(t *testing.T) {
	s := newStore(tmpDBPath)
	t.Cleanup(s.Destroy)

	var (
		msg1 = "test_value_1"
		msg2 = "test_value_2"
	)

	assert.NoError(t, s.Insert(defaultTopic, []byte(msg1)))
	assert.NoError(t, s.Insert(defaultTopic, []byte(msg2)))

	val, offset, err := s.GetNext(defaultTopic)
	assert.NoError(t, err)
	assert.Equal(t, msg1, string(val))

	assert.NoError(t, s.Nack(defaultTopic, offset))

	val, _, err = s.GetNext(defaultTopic)
	assert.NoError(t, err)
	assert.Equal(t, msg1, string(val))
}

// Nack
func TestBack(t *testing.T) {
	s := newStore(tmpDBPath)
	t.Cleanup(s.Destroy)

	assert.NoError(t, s.Insert(defaultTopic, []byte("test_value_1")))

	_, offset, err := s.GetNext(defaultTopic)
	assert.NoError(t, err)

	assert.NoError(t, s.Back(defaultTopic, offset))
}

func TestBackSentBack(t *testing.T) {
	s := newStore(tmpDBPath)
	t.Cleanup(s.Destroy)

	var (
		msg1 = []byte("test_value_1")
		msg2 = []byte("test_value_2")
	)

	assert.NoError(t, s.Insert(defaultTopic, msg1))
	assert.NoError(t, s.Insert(defaultTopic, msg2))

	_, offset, err := s.GetNext(defaultTopic)
	assert.NoError(t, err)

	assert.NoError(t, s.Back(defaultTopic, offset))

	v, _, err := s.GetNext(defaultTopic)
	assert.NoError(t, err)
	assert.Equal(t, msg2, v)
}

// Close
func TestClose(t *testing.T) {
	// TODO
}
