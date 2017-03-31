package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestMunge(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tests")
}

var _ = Describe("", func() {

	type MungeCase struct {
		InputFile  string
		OutputFile string
	}

	DescribeTable("munging the ftstate file",
		func(c MungeCase) {
			input, err := os.Open(fmt.Sprintf("testdata/%s.tfstate", c.InputFile))
			Expect(err).NotTo(HaveOccurred())
			defer input.Close()

			outputFile, err := os.Open(fmt.Sprintf("testdata/%s.tfstate", c.OutputFile))
			Expect(err).NotTo(HaveOccurred())
			defer outputFile.Close()

			expected, err := ioutil.ReadAll(outputFile)
			Expect(err).NotTo(HaveOccurred())

			out := &bytes.Buffer{}
			err = munge(input, out)
			Expect(err).NotTo(HaveOccurred())

			Expect(out.String()).To(Equal(string(expected)))
		},
		Entry("dev sample", MungeCase{InputFile: "input1", OutputFile: "output1"}),
		Entry("already done, shouldn't be changed", MungeCase{InputFile: "already_done", OutputFile: "already_done"}),
	)
})
