package memdb

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/lindb/series/field"
	"github.com/lindb/lindb/tsdb/tblstore/metricsdata"
)

func TestMetricStore_GetOrCreateTStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mStoreInterface := newMetricStore()
	mStore := mStoreInterface.(*metricStore)
	tStore, size := mStore.GetOrCreateTStore(uint32(10))
	assert.NotNil(t, tStore)
	assert.True(t, size > 0)
	tStore2, size := mStore.GetOrCreateTStore(uint32(10))
	assert.Zero(t, size)
	assert.Equal(t, tStore, tStore2)
}

func TestMetricStore_AddField(t *testing.T) {
	mStoreInterface := newMetricStore()
	mStore := mStoreInterface.(*metricStore)
	mStoreInterface.AddField(1, field.SumField)
	mStoreInterface.AddField(1, field.SumField)
	mStoreInterface.AddField(2, field.MinField)
	assert.Len(t, mStore.fields, 2)
	assert.Equal(t, field.Meta{ID: 1, Type: field.SumField}, mStore.fields[0])
	assert.Equal(t, field.Meta{ID: 2, Type: field.MinField}, mStore.fields[1])
}

func TestMetricStore_SetTimestamp(t *testing.T) {
	mStoreInterface := newMetricStore()
	mStore := mStoreInterface.(*metricStore)
	mStoreInterface.SetTimestamp(1, 10)
	slotRange, _ := mStore.families.GetRange(1)
	start, end := slotRange.getRange()
	assert.Equal(t, uint16(10), start)
	assert.Equal(t, uint16(10), end)
	mStoreInterface.SetTimestamp(1, 5)
	slotRange, _ = mStore.families.GetRange(1)
	start, end = slotRange.getRange()
	assert.Equal(t, uint16(5), start)
	assert.Equal(t, uint16(10), end)
	fmt.Println(start, end)
	mStoreInterface.SetTimestamp(1, 50)
	slotRange, _ = mStore.families.GetRange(1)
	start, end = slotRange.getRange()
	assert.Equal(t, uint16(5), start)
	assert.Equal(t, uint16(50), end)

	mStoreInterface.SetTimestamp(2, 50)
	mStoreInterface.SetTimestamp(4, 50)
	mStoreInterface.SetTimestamp(3, 50)
}

func TestMetricStore_FlushMetricsDataTo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
		flushFunc = flush
	}()

	flusher := metricsdata.NewMockFlusher(ctrl)

	mStoreInterface := newMetricStore()
	mStore := mStoreInterface.(*metricStore)
	tStore := NewMocktStoreINTF(ctrl)
	mStore.Put(10, tStore)

	// case 1: family time not exist
	err := mStoreInterface.FlushMetricsDataTo(flusher, flushContext{familyID: 1})
	assert.NoError(t, err)
	// case 2: field not exist
	mStoreInterface.SetTimestamp(1, 10)
	err = mStoreInterface.FlushMetricsDataTo(flusher, flushContext{familyID: 1})
	assert.NoError(t, err)
	// case 3: flush success
	mStoreInterface.AddField(1, field.SumField)
	mStoreInterface.AddField(2, field.MinField)
	gomock.InOrder(
		flusher.EXPECT().FlushFieldMetas(gomock.Any()),
		tStore.EXPECT().FlushSeriesTo(gomock.Any(), gomock.Any()),
		flusher.EXPECT().FlushSeries(uint32(10)),
		flusher.EXPECT().FlushMetric(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil),
	)
	err = mStoreInterface.FlushMetricsDataTo(flusher, flushContext{familyID: 1})
	assert.NoError(t, err)
	// case 4: flush err
	flushFunc = func(flusher metricsdata.Flusher, flushCtx flushContext, key uint32, value tStoreINTF) error {
		return fmt.Errorf("err")
	}
	gomock.InOrder(
		flusher.EXPECT().FlushFieldMetas(gomock.Any()),
	)
	err = mStoreInterface.FlushMetricsDataTo(flusher, flushContext{familyID: 1})
	assert.Error(t, err)
}
