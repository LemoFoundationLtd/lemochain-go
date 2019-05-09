package consensus

import "errors"

var (
	ErrBlockNotExist            = errors.New("block not exist in local")
	ErrIgnoreConfirm            = errors.New("confirms is enough")
	ErrExistBlock               = errors.New("block exist in local")
	ErrInvalidBlock             = errors.New("invalid block")
	ErrSaveBlock                = errors.New("save block to db error")
	ErrSaveAccount              = errors.New("save account error")
	ErrVerifyHeaderFailed       = errors.New("verify block's header error")
	ErrVerifyBlockFailed        = errors.New("verify block error")
	ErrSnapshotIsNil            = errors.New("local deputy nodes snapshot is nil")
	ErrInvalidConfirmSigner     = errors.New("invalid confirm signer")
	ErrInvalidSignedConfirmInfo = errors.New("invalid signed data of confirm info")
	ErrMineGenesis              = errors.New("can not mine genesis block")
	ErrNotDeputy                = errors.New("not a deputy address in specific height")
	ErrSetStableBlockToDB       = errors.New("set stable block to db error")
	ErrNoTermReward             = errors.New("reward value has not been set")
)
