package blockchain

import (
	"context"

	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/net/http/httperror"
	"github.com/bytom/net/http/httpjson"
	log "github.com/sirupsen/logrus"
)

func init() {
	//Error code 050 represents alias of key duplicated
	errorFormatter.Errors[pseudohsm.ErrDuplicateKeyAlias] = httperror.Info{400, "BTM050", "Alias already exists"}
	//Error code 801 represents query request format error
	errorFormatter.Errors[pseudohsm.ErrInvalidAfter] = httperror.Info{400, "BTM801", "Invalid `after` in query"}
	//Error code 802 represents query reponses too many
	errorFormatter.Errors[pseudohsm.ErrTooManyAliasesToList] = httperror.Info{400, "BTM802", "Too many aliases to list"}
}

func (a *BlockchainReactor) pseudohsmCreateKey(ctx context.Context, in struct{ Alias, Password string }) (result *pseudohsm.XPub, err error) {
	return a.hsm.XCreate(in.Alias, in.Password)
}

func (a *BlockchainReactor) pseudohsmListKeys(ctx context.Context, query requestQuery) (page, error) {
	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize // defGenericPageSize = 100
	}

	xpubs, after, err := a.hsm.ListKeys(query.After, limit)
	if err != nil {
		return page{}, err
	}

	var items []interface{}
	for _, xpub := range xpubs {
		items = append(items, xpub)
	}

	query.After = after

	return page{
		Items:    httpjson.Array(items),
		LastPage: len(xpubs) < limit,
		Next:     query,
	}, nil
}

func (a *BlockchainReactor) pseudohsmDeleteKey(ctx context.Context, x struct {
	Password string
	XPub     chainkd.XPub `json:"xpubs"`
}) error {
	return a.hsm.XDelete(x.XPub, x.Password)
}

func (a *BlockchainReactor) pseudohsmSignTemplates(ctx context.Context, x struct {
	Auth string
	Txs  []*txbuilder.Template `json:"transactions"`
	XPub chainkd.XPub          `json:"xpubs"`
	XPrv chainkd.XPrv          `json:"xprv"`
}) interface{} {
	resp := make([]interface{}, len(x.Txs))
	var err error
	for _, tx := range x.Txs {
		if x.Auth == "" {
			err = txbuilder.Sign(context.Background(), tx, []chainkd.XPub{x.XPrv.XPub()}, "", func(_ context.Context, _ chainkd.XPub, path [][]byte, data [32]byte, _ string) ([]byte, error) {
				derived := x.XPrv.Derive(path)
				return derived.Sign(data[:]), nil
			})
		} else {
			err = txbuilder.Sign(ctx, tx, []chainkd.XPub{x.XPub}, x.Auth, a.pseudohsmSignTemplate)
		}

		log.WithFields(log.Fields{"tx": tx, "build err": err}).Info("After sign transaction.")
		if err != nil {
			info := errorFormatter.Format(err)
			resp = append(resp, info)
		} else {
			resp = append(resp, tx)
		}
	}
	log.WithField("resp", resp).Info("Sign Transaction complete.")
	return resp
}

func (a *BlockchainReactor) pseudohsmSignTemplate(ctx context.Context, xpub chainkd.XPub, path [][]byte, data [32]byte, password string) ([]byte, error) {
	sigBytes, err := a.hsm.XSign(xpub, path, data[:], password)
	if err == pseudohsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}

func (a *BlockchainReactor) pseudohsmResetPassword(ctx context.Context, x struct {
	OldPassword string
	NewPassword string
	XPub        chainkd.XPub `json:"xpubs"`
}) error {
	return a.hsm.ResetPassword(x.XPub, x.OldPassword, x.NewPassword)
}
