package runtime_test

import (
	"fmt"

	"github.com/LemoFoundationLtd/lemochain-core/chain/vm/runtime"
	"github.com/LemoFoundationLtd/lemochain-core/common"
)

func ExampleExecute() {
	ret, err := runtime.Execute(common.FromHex("6060604052600a8060106000396000f360606040526008565b00"), nil, nil)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(ret)
	// Output:
	// [96 96 96 64 82 96 8 86 91 0]
}
