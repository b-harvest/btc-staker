package e2etest

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"

	"github.com/babylonlabs-io/btc-staker/itest/containers"
	"github.com/stretchr/testify/require"
)

var (
	startTimeout = 30 * time.Second
)

type CreateWalletResponse struct {
	Name    string `json:"name"`
	Warning string `json:"warning"`
}

type GenerateBlockResponse struct {
	// address of the recipient of rewards
	Address string `json:"address"`
	// blocks generated
	Blocks []string `json:"blocks"`
}

type BitcoindTestHandler struct {
	t *testing.T
	m *containers.Manager
}

func NewBitcoindHandler(t *testing.T, m *containers.Manager) *BitcoindTestHandler {
	return &BitcoindTestHandler{
		t: t,
		m: m,
	}
}

func (h *BitcoindTestHandler) Start() *dockertest.Resource {
	tempPath, err := os.MkdirTemp("", "bitcoind-staker-test-*")
	require.NoError(h.t, err)

	h.t.Cleanup(func() {
		_ = os.RemoveAll(tempPath)
	})

	bitcoinResource, err := h.m.RunBitcoindResource(h.t, tempPath)
	require.NoError(h.t, err)

	h.t.Cleanup(func() {
		_ = h.m.ClearResources()
	})

	require.Eventually(h.t, func() bool {
		_, err := h.GetBlockCount()
		h.t.Logf("failed to get block count: %v", err)
		return err == nil
	}, startTimeout, 500*time.Millisecond, "bitcoind did not start")

	return bitcoinResource
}

func (h *BitcoindTestHandler) GetBlockCount() (int, error) {
	buff, _, err := h.m.ExecBitcoindCliCmd(h.t, []string{"getblockcount"})
	if err != nil {
		return 0, err
	}

	buffStr := buff.String()

	parsedBuffStr := strings.TrimSuffix(buffStr, "\n")

	num, err := strconv.Atoi(parsedBuffStr)
	if err != nil {
		return 0, err
	}

	return num, nil
}

func (h *BitcoindTestHandler) CreateWallet(walletName string, passphrase string) *CreateWalletResponse {
	// last false on the list will create legacy wallet. This is needed, as currently
	// we are signing all taproot transactions by dumping the private key and signing it
	// on app level. Descriptor wallets do not allow dumping private keys.
	buff, _, err := h.m.ExecBitcoindCliCmd(h.t, []string{"createwallet", walletName, "false", "false", passphrase})
	require.NoError(h.t, err)

	var response CreateWalletResponse
	err = json.Unmarshal(buff.Bytes(), &response)
	require.NoError(h.t, err)

	return &response
}

func (h *BitcoindTestHandler) GenerateBlocks(count int) *GenerateBlockResponse {
	buff, _, err := h.m.ExecBitcoindCliCmd(h.t, []string{"-generate", fmt.Sprintf("%d", count)})
	require.NoError(h.t, err)

	var response GenerateBlockResponse
	err = json.Unmarshal(buff.Bytes(), &response)
	require.NoError(h.t, err)

	return &response
}
