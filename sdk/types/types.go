// Package types mirrors hyperliquid-python-sdk/hyperliquid/utils/types.py.
package types

import (
	"fmt"
	"strings"
)

// Cloid corresponds to Python:
// hyperliquid.utils.types.Cloid
type Cloid struct {
	raw string
}

// NewCloid corresponds to Python:
// hyperliquid.utils.types.Cloid.from_str
func NewCloid(raw string) (Cloid, error) {
	c := Cloid{raw: raw}
	if err := c.validate(); err != nil {
		return Cloid{}, err
	}
	return c, nil
}

// CloidFromInt corresponds to Python:
// hyperliquid.utils.types.Cloid.from_int
func CloidFromInt(v uint64) Cloid {
	return Cloid{raw: fmt.Sprintf("0x%032x", v)}
}

func (c Cloid) validate() error {
	if !strings.HasPrefix(c.raw, "0x") {
		return fmt.Errorf("cloid is not a hex string")
	}
	if len(c.raw[2:]) != 32 {
		return fmt.Errorf("cloid is not 16 bytes")
	}
	return nil
}

// String corresponds to Python:
// hyperliquid.utils.types.Cloid.__str__
func (c Cloid) String() string {
	return c.raw
}

// ToRaw corresponds to Python:
// hyperliquid.utils.types.Cloid.to_raw
func (c Cloid) ToRaw() string {
	return c.raw
}

// Meta mirrors hyperliquid.utils.types.Meta.
type Meta struct {
	Universe []AssetInfo `json:"universe"`
}

// AssetInfo mirrors hyperliquid.utils.types.AssetInfo.
type AssetInfo struct {
	Name       string `json:"name"`
	SzDecimals int    `json:"szDecimals"`
}

// SpotMeta mirrors hyperliquid.utils.types.SpotMeta.
type SpotMeta struct {
	Universe []SpotAssetInfo `json:"universe"`
	Tokens   []SpotTokenInfo `json:"tokens"`
}

// SpotAssetInfo mirrors hyperliquid.utils.types.SpotAssetInfo.
type SpotAssetInfo struct {
	Name        string `json:"name"`
	Tokens      []int  `json:"tokens"`
	Index       int    `json:"index"`
	IsCanonical bool   `json:"isCanonical"`
}

// SpotTokenInfo mirrors hyperliquid.utils.types.SpotTokenInfo.
type SpotTokenInfo struct {
	Name        string  `json:"name"`
	SzDecimals  int     `json:"szDecimals"`
	WeiDecimals int     `json:"weiDecimals"`
	Index       int     `json:"index"`
	TokenID     string  `json:"tokenId"`
	IsCanonical bool    `json:"isCanonical"`
	EVMContract *string `json:"evmContract"`
	FullName    *string `json:"fullName"`
}

// BuilderInfo mirrors hyperliquid.utils.types.BuilderInfo.
type BuilderInfo struct {
	B string `json:"b"`
	F int    `json:"f"`
}

// PerpDexSchemaInput mirrors hyperliquid.utils.types.PerpDexSchemaInput.
type PerpDexSchemaInput struct {
	FullName        string
	CollateralToken int
	OracleUpdater   *string
}
