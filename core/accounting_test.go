package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBalance_Credit(t *testing.T) {
	mid := ManifestID("some manifestID")
	balances := NewBalances(5 * time.Second)
	b := NewBalance(mid, balances)

	assert := assert.New(t)

	b.Credit(big.NewRat(5, 1))
	assert.Zero(big.NewRat(5, 1).Cmp(balances.Balance(mid)))

	b.Credit(big.NewRat(-5, 1))
	assert.Zero(big.NewRat(0, 1).Cmp(balances.Balance(mid)))

	b.Credit(big.NewRat(0, 1))
	assert.Zero(big.NewRat(0, 1).Cmp(balances.Balance(mid)))
}

func TestBalance_StageUpdate(t *testing.T) {
	mid := ManifestID("some manifestID")
	balances := NewBalances(5 * time.Second)
	b := NewBalance(mid, balances)

	assert := assert.New(t)

	// Test existing credit > minimum credit
	b.Credit(big.NewRat(2, 1))
	numTickets, newCredit, existingCredit := b.StageUpdate(big.NewRat(1, 1), nil)
	assert.Equal(0, numTickets)
	assert.Zero(big.NewRat(0, 1).Cmp(newCredit))
	assert.Zero(big.NewRat(2, 1).Cmp(existingCredit))
	assert.Zero(big.NewRat(0, 1).Cmp(balances.Balance(mid)))

	// Test existing credit = minimum credit
	b.Credit(big.NewRat(2, 1))
	numTickets, newCredit, existingCredit = b.StageUpdate(big.NewRat(2, 1), nil)
	assert.Equal(0, numTickets)
	assert.Zero(big.NewRat(0, 1).Cmp(newCredit))
	assert.Zero(big.NewRat(2, 1).Cmp(existingCredit))
	assert.Zero(big.NewRat(0, 1).Cmp(balances.Balance(mid)))

	// Test exact number of tickets covers new credit
	b.Credit(big.NewRat(1, 1))
	numTickets, newCredit, existingCredit = b.StageUpdate(big.NewRat(5, 1), big.NewRat(1, 1))
	assert.Equal(4, numTickets)
	assert.Zero(big.NewRat(4, 1).Cmp(newCredit))
	assert.Zero(big.NewRat(1, 1).Cmp(existingCredit))
	assert.Zero(big.NewRat(0, 1).Cmp(balances.Balance(mid)))

	// Test non-exact number of tickets covers new credit
	b.Credit(big.NewRat(1, 4))
	numTickets, newCredit, existingCredit = b.StageUpdate(big.NewRat(2, 1), big.NewRat(1, 1))
	assert.Equal(2, numTickets)
	assert.Zero(big.NewRat(2, 1).Cmp(newCredit))
	assert.Zero(big.NewRat(1, 4).Cmp(existingCredit))
	assert.Zero(big.NewRat(0, 1).Cmp(balances.Balance(mid)))

	// Test negative existing credit
	b.Credit(big.NewRat(-5, 1))
	numTickets, newCredit, existingCredit = b.StageUpdate(big.NewRat(2, 1), big.NewRat(1, 1))
	assert.Equal(7, numTickets)
	assert.Zero(big.NewRat(7, 1).Cmp(newCredit))
	assert.Zero(big.NewRat(-5, 1).Cmp(existingCredit))
	assert.Zero(big.NewRat(0, 1).Cmp(balances.Balance(mid)))

	// Test no existing credit
	numTickets, newCredit, existingCredit = b.StageUpdate(big.NewRat(2, 1), big.NewRat(1, 1))
	assert.Equal(2, numTickets)
	assert.Zero(big.NewRat(2, 1).Cmp(newCredit))
	assert.Zero(big.NewRat(0, 1).Cmp(existingCredit))
	assert.Zero(big.NewRat(0, 1).Cmp(balances.Balance(mid)))
}

func TestBalance_Clear(t *testing.T) {
	mid := ManifestID("some manifestID")
	balances := NewBalances(5 * time.Second)
	b := NewBalance(mid, balances)

	assert := assert.New(t)

	// Test non-nil key
	b.Credit(big.NewRat(5, 1))
	b.Clear()
	assert.Nil(balances.balances[mid])

	// Test nil key
	b.Clear()
	assert.Nil(balances.balances[mid])
}

func TestEmptyBalances_ReturnsZeroedValues(t *testing.T) {
	mid := ManifestID("some manifest id")
	b := NewBalances(5 * time.Second)
	assert := assert.New(t)

	assert.Nil(b.Balance(mid))
	assert.Nil(b.balances[mid])
}

func TestCredit_ReturnsNewCreditBalance(t *testing.T) {
	mid := ManifestID("some manifest id")
	b := NewBalances(5 * time.Second)
	assert := assert.New(t)
	amount := big.NewRat(100, 1)

	b.Credit(mid, amount)
	assert.Zero(b.Balance(mid).Cmp(amount))
}

func TestDebitAfterCredit_SameAmount_ReturnsZero(t *testing.T) {
	mid := ManifestID("some manifest id")
	b := NewBalances(5 * time.Second)
	assert := assert.New(t)
	amount := big.NewRat(100, 1)

	b.Credit(mid, amount)
	assert.Zero(b.Balance(mid).Cmp(amount))

	b.Debit(mid, amount)
	assert.Zero(b.Balance(mid).Cmp(big.NewRat(0, 1)))
}

func TestDebitHalfOfCredit_ReturnsHalfOfCredit(t *testing.T) {
	mid := ManifestID("some manifest id")
	b := NewBalances(5 * time.Second)
	assert := assert.New(t)
	credit := big.NewRat(100, 1)
	debit := big.NewRat(50, 1)
	b.Credit(mid, credit)
	assert.Zero(b.Balance(mid).Cmp(credit))

	b.Debit(mid, debit)
	assert.Zero(b.Balance(mid).Cmp(debit))
}

func TestReserve(t *testing.T) {
	assert := assert.New(t)

	mid := ManifestID("some manifest id")
	b := NewBalances(5 * time.Second)

	// Test when entry is nil
	assert.Zero(big.NewRat(0, 1).Cmp(b.Reserve(mid)))
	assert.Zero(big.NewRat(0, 1).Cmp(b.Balance(mid)))

	// Test when entry is non-nil
	b.Credit(mid, big.NewRat(5, 1))
	assert.Zero(big.NewRat(5, 1).Cmp(b.Reserve(mid)))
	assert.Zero(big.NewRat(0, 1).Cmp(b.Balance(mid)))

	// Test when amount is negative
	b.Debit(mid, big.NewRat(5, 1))
	assert.Zero(big.NewRat(-5, 1).Cmp(b.Reserve(mid)))
	assert.Zero(big.NewRat(0, 1).Cmp(b.Balance(mid)))
}

func TestBalancesCleanup(t *testing.T) {
	b := NewBalances(5 * time.Second)
	assert := assert.New(t)

	// Set up two mids
	// One we will update after 2*time.Seconds
	// The other one we will not update before timeout
	// This should run clean only the second
	mid1 := ManifestID("First MID")
	mid2 := ManifestID("Second MID")
	// Start cleanup loop
	go b.StartCleanup()
	defer b.StopCleanup()

	// Fund balances
	credit := big.NewRat(100, 1)
	b.Credit(mid1, credit)
	b.Credit(mid2, credit)
	assert.Zero(b.Balance(mid1).Cmp(credit))
	assert.Zero(b.Balance(mid2).Cmp(credit))

	time.Sleep(2 * time.Second)
	b.Credit(mid1, credit)
	assert.Zero(b.Balance(mid1).Cmp(big.NewRat(200, 1)))

	time.Sleep(4 * time.Second)

	// Balance for mid1 should still be 200/1
	assert.NotNil(b.Balance(mid1))
	assert.Zero(b.Balance(mid1).Cmp(big.NewRat(200, 1)))
	// Balance for mid2 should be cleaned
	assert.Nil(b.Balance(mid2))

	time.Sleep(5 * time.Second)
	// Now balance for mid1 should be cleaned as well
	assert.Nil(b.Balance(mid1))
}
