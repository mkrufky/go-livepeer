package pm

import (
	"math/big"
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setTime is a helper to set the time during tests
var setTime = func(time int64) {
	unixNow = func() int64 {
		return time
	}
}

// increaseTime is a helper to increase the time during tests
var increaseTime = func(sec int64) {
	time := unixNow()
	setTime(time + sec)
}

func TestMaxFloat(t *testing.T) {
	claimant, b, smgr, rm, em := senderMonitorFixture()
	addr := RandAddress()
	smgr.info[addr] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       big.NewInt(500),
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr] = big.NewInt(100)
	rm.transcoderPoolSize = big.NewInt(50)
	sm := NewSenderMonitor(claimant, b, smgr, rm, 5*time.Minute, 3600, em)
	sm.Start()
	defer sm.Stop()

	assert := assert.New(t)

	// Test ClaimedReserve() error
	smgr.err = errors.New("ClaimedReserve error")

	_, err := sm.MaxFloat(RandAddress())
	assert.EqualError(err, "ClaimedReserve error")

	// Test value cached

	smgr.err = nil
	reserve := new(big.Int).Sub(new(big.Int).Div(smgr.info[addr].Reserve, rm.transcoderPoolSize), smgr.claimedReserve[addr])

	mf, err := sm.MaxFloat(addr)
	assert.Equal(reserve, mf)
}

func TestSubFloat(t *testing.T) {
	claimant, b, smgr, rm, em := senderMonitorFixture()
	addr := RandAddress()
	smgr.info[addr] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       big.NewInt(500),
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr] = big.NewInt(100)
	rm.transcoderPoolSize = big.NewInt(50)
	sm := NewSenderMonitor(claimant, b, smgr, rm, 5*time.Minute, 3600, em)
	sm.Start()
	defer sm.Stop()

	assert := assert.New(t)
	require := require.New(t)

	reserve := new(big.Int).Sub(new(big.Int).Div(smgr.info[addr].Reserve, rm.transcoderPoolSize), smgr.claimedReserve[addr])

	amount := big.NewInt(5)
	sm.SubFloat(addr, amount)
	mf, err := sm.MaxFloat(addr)
	require.Nil(err)
	assert.Equal(new(big.Int).Sub(reserve, amount), mf)

	assert.True(em.AcceptErr(claimant))

	em.acceptable = false

	sm.SubFloat(addr, amount)
	assert.Nil(err)

	mf, err = sm.MaxFloat(addr)
	require.Nil(err)
	assert.Equal(
		new(big.Int).Sub(reserve, new(big.Int).Mul(amount, big.NewInt(2))),
		mf,
	)

	// Test resetting errCount
	assert.True(em.AcceptErr(claimant))
}
func TestAddFloat(t *testing.T) {
	claimant, b, smgr, rm, em := senderMonitorFixture()
	addr := RandAddress()
	smgr.info[addr] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       big.NewInt(500),
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr] = big.NewInt(100)
	rm.transcoderPoolSize = big.NewInt(1)
	sm := NewSenderMonitor(claimant, b, smgr, rm, 5*time.Minute, 3600, em)
	sm.Start()
	defer sm.Stop()

	assert := assert.New(t)
	require := require.New(t)

	// Test ClaimedReserve() error
	smgr.err = errors.New("ClaimedReserve error")

	em.acceptable = false

	sm.SubFloat(addr, big.NewInt(10))
	err := sm.AddFloat(addr, big.NewInt(10))
	assert.EqualError(err, "ClaimedReserve error")

	// Test value not cached and insufficient pendingAmount error
	smgr.err = nil
	reserve := new(big.Int).Sub(new(big.Int).Div(smgr.info[addr].Reserve, rm.transcoderPoolSize), smgr.claimedReserve[addr])

	amount := big.NewInt(20)
	err = sm.AddFloat(addr, amount)
	assert.EqualError(err, "cannot subtract from insufficient pendingAmount")

	// Test value cached and no pendingAmount error

	sm.SubFloat(addr, amount)

	err = sm.AddFloat(addr, amount)
	assert.Nil(err)

	mf, err := sm.MaxFloat(addr)
	require.Nil(err)
	assert.Equal(mf, reserve)

	// Test cached value update
	smgr.info[addr].Reserve = big.NewInt(1000)
	reserve = new(big.Int).Sub(new(big.Int).Div(smgr.info[addr].Reserve, rm.transcoderPoolSize), smgr.claimedReserve[addr])

	sm.SubFloat(addr, amount)

	assert.True(em.AcceptErr(claimant))

	em.acceptable = false
	err = sm.AddFloat(addr, amount)
	assert.Nil(err)

	mf, err = sm.MaxFloat(addr)
	require.Nil(err)
	assert.Equal(reserve, mf)
	assert.True(em.acceptable)
}

func TestQueueTicketAndSignalMaxFloat(t *testing.T) {
	claimant, b, smgr, rm, em := senderMonitorFixture()
	addr := RandAddress()
	smgr.info[addr] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       big.NewInt(5000),
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr] = big.NewInt(100)
	sm := NewSenderMonitor(claimant, b, smgr, rm, 5*time.Minute, 3600, em)
	sm.Start()
	defer sm.Stop()

	assert := assert.New(t)
	require := require.New(t)

	// Test queue ticket

	sm.QueueTicket(addr, defaultSignedTicket(uint32(0)))

	sm.SubFloat(addr, big.NewInt(5))

	qc := &queueConsumer{}
	go qc.Wait(1, sm)

	err := sm.AddFloat(addr, big.NewInt(5))
	require.Nil(err)

	time.Sleep(time.Millisecond * 20)
	tickets := qc.Redeemable()
	assert.Equal(1, len(tickets))
	assert.Equal(uint32(0), tickets[0].SenderNonce)

	// Test queue tickets from multiple senders

	addr2 := RandAddress()
	smgr.info[addr2] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       big.NewInt(5000),
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr2] = big.NewInt(100)

	sm.QueueTicket(addr, defaultSignedTicket(uint32(2)))
	sm.QueueTicket(addr2, defaultSignedTicket(uint32(3)))

	sm.SubFloat(addr, big.NewInt(5))
	sm.SubFloat(addr2, big.NewInt(5))

	qc = &queueConsumer{}
	go qc.Wait(2, sm)

	err = sm.AddFloat(addr2, big.NewInt(5))
	require.Nil(err)
	err = sm.AddFloat(addr, big.NewInt(5))
	require.Nil(err)

	time.Sleep(time.Millisecond * 20)
	// Order of tickets should reflect order that AddFloat()
	// was called
	tickets = qc.Redeemable()
	assert.Equal(2, len(tickets))
	assert.Equal(uint32(3), tickets[0].SenderNonce)
	assert.Equal(uint32(2), tickets[1].SenderNonce)
}

func TestCleanup(t *testing.T) {
	claimant, b, smgr, rm, em := senderMonitorFixture()
	sm := NewSenderMonitor(claimant, b, smgr, rm, 5*time.Minute, 3600, em)
	sm.Start()
	defer sm.Stop()

	assert := assert.New(t)
	require := require.New(t)

	setTime(0)

	// TODO: Test ticker?

	// Test clean up
	addr1 := RandAddress()
	addr2 := RandAddress()
	smgr.info[addr1] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       big.NewInt(500),
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr1] = big.NewInt(100)
	smgr.info[addr2] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       big.NewInt(500),
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr2] = big.NewInt(100)

	// Set lastAccess
	_, err := sm.MaxFloat(addr1)
	require.Nil(err)
	_, err = sm.MaxFloat(addr2)
	require.Nil(err)

	increaseTime(10)

	// Change stub SenderManager values
	// SenderMonitor should no longer use cached values
	// since they have been cleaned up
	sm.(*senderMonitor).cleanup()
	smgr.Clear(addr1)
	smgr.Clear(addr2)
	assert.Nil(smgr.info[addr1])
	assert.Nil(smgr.claimedReserve[addr1])
	assert.Nil(smgr.info[addr2])
	assert.Nil(smgr.claimedReserve[addr2])

	reserve2 := big.NewInt(1000)
	smgr.info[addr1] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       reserve2,
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr1] = big.NewInt(100)
	smgr.info[addr2] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       reserve2,
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr2] = big.NewInt(100)

	mf1, err := sm.MaxFloat(addr1)
	require.Nil(err)
	mf2, err := sm.MaxFloat(addr2)
	require.Nil(err)

	expectedAlloc := new(big.Int).Sub(new(big.Int).Div(smgr.info[addr1].Reserve, rm.transcoderPoolSize), smgr.claimedReserve[addr1])

	assert.Equal(expectedAlloc, mf1)
	assert.Equal(expectedAlloc, mf2)

	// Test clean up after excluding items
	// with updated lastAccess due to MaxFloat()

	// Update lastAccess for addr1
	increaseTime(4)
	_, err = sm.MaxFloat(addr1)
	require.Nil(err)

	increaseTime(1)

	// Change stub broker value
	// SenderMonitor should:
	// - Use cached value for addr1 because it was accessed recently via MaxFloat()
	// - Use new value for addr2 because it was cleaned up
	reserve3 := big.NewInt(100)
	smgr.info[addr2].Reserve = reserve3

	sm.(*senderMonitor).cleanup()

	mf1, err = sm.MaxFloat(addr1)
	require.Nil(err)
	mf2, err = sm.MaxFloat(addr2)
	require.Nil(err)

	expectedAlloc2 := new(big.Int).Sub(new(big.Int).Div(smgr.info[addr2].Reserve, rm.transcoderPoolSize), smgr.claimedReserve[addr2])
	assert.Equal(expectedAlloc, mf1)
	assert.Equal(expectedAlloc2, mf2)

	// Test clean up excluding items
	// with updated lastAccess due to AddFloat()

	// Update lastAccess for addr2
	increaseTime(4)
	err = sm.AddFloat(addr2, big.NewInt(0))
	require.Nil(err)

	increaseTime(1)

	// Change stub broker value
	// SenderMonitor should:
	// - Use new value for addr1 because it was cleaned up
	// - Use cached value for addr2 because it was accessed recently via AddFloat()
	reserve4 := big.NewInt(101)
	smgr.info[addr1].Reserve = reserve4

	sm.(*senderMonitor).cleanup()

	mf1, err = sm.MaxFloat(addr1)
	require.Nil(err)
	mf2, err = sm.MaxFloat(addr2)
	require.Nil(err)

	expectedAlloc3 := new(big.Int).Sub(new(big.Int).Div(smgr.info[addr1].Reserve, rm.transcoderPoolSize), smgr.claimedReserve[addr1])

	assert.Equal(expectedAlloc3, mf1)
	assert.Equal(expectedAlloc2, mf2)

	// Test clean up excluding items
	// with updated lastAccess due to SubFloat()

	// Update lastAccess for addr1
	increaseTime(4)
	sm.SubFloat(addr1, big.NewInt(0))

	increaseTime(1)

	// Change stub broker value
	// SenderMonitor should:
	// - Use cached value for addr1 because it was accessed recently via SubFloat()
	// - Use new value for addr2 because it was cleaned up
	reserve5 := big.NewInt(999)
	smgr.info[addr2].Reserve = reserve5

	sm.(*senderMonitor).cleanup()

	mf1, err = sm.MaxFloat(addr1)
	require.Nil(err)
	mf2, err = sm.MaxFloat(addr2)
	require.Nil(err)

	expectedAlloc4 := new(big.Int).Sub(new(big.Int).Div(smgr.info[addr2].Reserve, rm.transcoderPoolSize), smgr.claimedReserve[addr2])
	assert.Equal(expectedAlloc3, mf1)
	assert.Equal(expectedAlloc4, mf2)
}

func TestReserveAlloc(t *testing.T) {
	assert := assert.New(t)
	claimant, b, smgr, rm, em := senderMonitorFixture()
	addr := RandAddress()
	smgr.info[addr] = &SenderInfo{
		Deposit:       big.NewInt(500),
		Reserve:       big.NewInt(5000),
		WithdrawBlock: big.NewInt(0),
		ReserveState:  NotFrozen,
		ThawRound:     big.NewInt(0),
	}
	smgr.claimedReserve[addr] = big.NewInt(100)
	sm := NewSenderMonitor(claimant, b, smgr, rm, 5*time.Minute, 3600, em).(*senderMonitor)

	// test GetSenderInfo error
	smgr.err = errors.New("GetSenderInfo error")
	_, err := sm.reserveAlloc(addr)
	assert.EqualError(err, smgr.err.Error())
	// test reserveAlloc correctly calculated
	smgr.err = nil
	expectedAlloc := new(big.Int).Sub(new(big.Int).Div(smgr.info[addr].Reserve, rm.transcoderPoolSize), smgr.claimedReserve[addr])
	alloc, err := sm.reserveAlloc(addr)
	assert.Nil(err)
	assert.Zero(expectedAlloc.Cmp(alloc))
}

func senderMonitorFixture() (ethcommon.Address, *stubBroker, *stubSenderManager, *stubRoundsManager, *stubErrorMonitor) {
	claimant := RandAddress()
	b := newStubBroker()
	smgr := newStubSenderManager()
	rm := &stubRoundsManager{
		transcoderPoolSize: big.NewInt(5),
	}
	em := &stubErrorMonitor{}
	return claimant, b, smgr, rm, em
}
