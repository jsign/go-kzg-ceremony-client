[![Go Report Card](https://goreportcard.com/badge/github.com/jsign/go-kzg-ceremony-client)](https://goreportcard.com/report/github.com/jsign/go-kzg-ceremony-client) ![test](https://github.com/jsign/go-kzg-ceremony-client/actions/workflows/test.yaml/badge.svg) ![releases](https://github.com/jsign/go-kzg-ceremony-client/actions/workflows/release.yaml/badge.svg)


# Ethereum EIP-4844 Powers of Tau ceremony client

This repository contains an implementation of a client to participate in the Powers of Tau ceremony for EIP-4844 in  Ethereum. This is a multi-party ceremony to generate an SRS needed for KZG commitments.

For _bls12-381_ elliptic curve operations such as group multiplication and pairings, the implementation uses the [gnark-crypto](https://github.com/ConsenSys/gnark-crypto) library ([audited Oct-2022](https://github.com/ConsenSys/gnark-crypto/blob/master/audit_oct2022.pdf)).

## Table of content
- [Ethereum EIP-4844 Powers of Tau ceremony client](#ethereum-eip-4844-powers-of-tau-ceremony-client)
  - [Table of content](#table-of-content)
  - [What are the Powers of Tau ceremony, EIP-4844 and KZG commitments?](#what-are-the-powers-of-tau-ceremony-eip-4844-and-kzg-commitments)
  - [Features](#features)
  - [I want to participate in the ceremony, how should I use this client?](#i-want-to-participate-in-the-ceremony-how-should-i-use-this-client)
    - [Step 1 - Get the `kzgcli` CLI command](#step-1---get-the-kzgcli-cli-command)
    - [Step 2 - Get your session-id keys](#step-2---get-your-session-id-keys)
    - [Step 3 - Contribute!](#step-3---contribute)
    - [Step 4 (optional) - Check that your contribution is in the new transcript](#step-4-optional---check-that-your-contribution-is-in-the-new-transcript)
  - [External entropy](#external-entropy)
  - [Offline contributions](#offline-contributions)
  - [Verify the current sequencer transcript ourselves](#verify-the-current-sequencer-transcript-ourselves)
  - [Tests and benchmarks](#tests-and-benchmarks)
  - [Side-effects of this ceremony client work](#side-effects-of-this-ceremony-client-work)
  - [Potential improvements](#potential-improvements)
  - [License](#license)

## What are the Powers of Tau ceremony, EIP-4844 and KZG commitments?
If you're confused about these terms, the best place to understand them better is the [official ceremony](https://ceremony.ethereum.org/) website which gives a high-level explanation of these concepts. It also has a useful FAQ section that digs a bit deeper into the details.

## Features
This client implementation has the following features:
- Supports the expected flow for contributing to the ceremony both with Github and Ethereum addresses. All clients should, at a minimum, implement this to contribute to the ceremony as expected.
- Supports a special command to pull the current transcript from the ceremony sequencer, and check locally that it's valid. This can be done at any point in the ceremony to check that the sequencer is being honest. No login/authentication is needed!
- The entropy source used to generate the contributions secret is at a minimum a [CSRNG](https://en.wikipedia.org/wiki/Cryptographically_secure_pseudorandom_number_generator) from the Go standard-library.
- It supports two opt-in sources of entropy which add entropy on top of the CSRNG:
  - Entropy generated by the [drand network](https://drand.love/) at the contribution point.
  - Entropy from an external REST API to pull entropy from an arbitrary source. This can be helpful for people contributing creative entropy sources.
- For each of the sub-ceremonies, a different secret is generated from the entropy sources as recommended (i.e: **not** use the same secret in sub-ceremonies)

Using external entropy **does not** interfere with contribution time. It's pulled before starting to ask for our turn to the sequencer, so drand and/or the REST API can't add a failure case or extra delays. This is important to contribute as fast as possible, and allow the sequencer to give the turn to another contributor!


## I want to participate in the ceremony, how should I use this client? 
Contributing to the ceremony is very easy, you only need at least one of:
- An Ethereum address that has sent at least 3 transactions at the Merge block number.
- A GitHub account that has a commit dated before 1 August 2022 00:00 UTC.

Currently, the ceremony sequencer is in testnet mode, but everything described in this section will be the same when the real ceremony starts.

### Step 1 - Get the `kzgcli` CLI command
The `kzgcli` command is the CLI of this ceremony client, you can get it in different ways:
- Compiling from source: For this, you'll need to have installed the `go` compiler. Pull this repo, and run `make build` or `go build -o kzgcli ./cmd/kzgcli`. This will leave the `kzgcli` binary in your current folder (you might want to `sudo mv kzgcli /usr/local/bin` or your PATH for convenience).
- Download binaries from the [releases section](https://github.com/jsign/go-kzg-ceremony-client/releases).


### Step 2 - Get your session-id keys
You'll need a `session-id` to participate in the ceremony. A `session-id` is a GUID string that you'll need to pass to the `kzgcli` CLI as a flag.

To get your `session-id` do the following:
- Open the [request_link](https://seq.ceremony.ethereum.org/auth/request_link) endpoint in your browser.
- You'll be presented with two links, one for Ethereum address participation and one for GitHub account participation. Open the corresponding link and follow the explained steps.
- In the end, you'll receive a JSON that has a `session_id` field with a value _similar to_ `504d898c-e975-4e13-9a48-4f8b95d754fb`. This string is your `session-id`, copy it to your clipboard.

Note that this step of the process is done on an external website unrelated to this ceremony client. This website is related to the sequencer which all clients target and is managed by the Ethereum Foundation.

If you got an error trying to get your `session-id`, it could be one of the following ones:
- `AuthErrorPayload::UserCreatedAfterDeadline`: your Ethereum address isn't matching the sequencer minimal conditions. Your Ethereum address should have sent at least 3 transactions at block 15537393. If that isn't true, you can't participate with this Ethereum address.
- `AuthErrorPayload::InvalidAuthCode`: your request link got stale. Start the login process from scratch.
- `AuthErrorPayload::UserAlreadyContributed`: you can only contribute once per GitHub account or Ethereum address.

### Step 3 - Contribute!

Optionally, you can first check the status of the lobby:
```bash
$ kzgcli status
Lobby size: 0
Number of contributions: 7915
Sequencer address: 0x5afE36d82dE8990B777f82651B96608Ec54d190d
```
This can provide some context around how many people are waiting for their turn to contribute and a sense of waiting times.

Contribute to the ceremony by running:
```
$ kzgcli contribute --session-id <paste-your-session-id>
Waiting for our turn to contribute...
It's our turn! Contributing...
Contribution ready, took 2.91s
Sending contribution...
Success!
```
That's it! Two files will appear in your current directory:
- `my_contribution.json` is exactly the contribution that was submitted to the sequencer.
- `contribution_receipt.json` is the receipt returned by the sequencer for your contribution.

If you want to leverage the optional external sources of entropy, you can provide some extra flags. Please check the _External entropy_ section below for more details and examples.

### Step 4 (optional) - Check that your contribution is in the new transcript
The sequencer has a public transcript that contains everyone that has participated correctly in the ceremony.
A friendly way is looking at the [ceremony website contributors list](https://ceremony.ethereum.org/#/record).

## External entropy
The `kzgcli contribute` command has two optional flags:
- `--drand`: if this flag is provided, the client will pull the latest entropy available from the [drand network](https://drand.love/), which will be mixed with the CSRNG source when contributing to the ceremony.
- `--urlrand <url>`: is a URL that the client will do a `GET` request, and use the returned body bytes as entropy to be mixed with the CSRNG source when contributing to the ceremony.

You can provide only one of these flags, or both at the same time:
```
$ kzgcli contribute --session-id <session-id> --drand --urlrand https://ihagopian.com
Pulling entropy from drand... Got it! (length: 32, round: 2578668)
Pulling entropy from https://ihagopian.com... Got it! (length: 52919)
Waiting for our turn to contribute...
It's our turn! Contributing...
Contribution ready, took 3.01s
Sending contribution...
Success!
```

If you want to understand in more detail how the external entropy is mixed with the CSRNG, please see [this code section](https://github.com/jsign/go-kzg-ceremony-client/blob/main/contribution/batchcontribution.go#L24-L35).

## Offline contributions
This section is only interesting if you're contributing from constrained environments.

Apart from conforming to the specification for the Powers of Tau protocol, participating in the ceremony involves interacting with the sequencer in a defined API flow. If you are contributing from a constraint environment (e.g: air-gapped or bandwidth constrained), you might be interested in narrowing down the contribution step independently from getting the state and sending the contribution.

The CLI tool provides an _offline_ subcommand:

- `kzgcli offline download-state <file-path>`: downloads the current state of the ceremony from the sequencer and saves it in a file.
- `kzgcli offline contribute <current-state-path> <contribution-path>`: opens a previously downloaded current state of the ceremony, makes the contribution and saves it in a new file.
- `kzgcli offline send-contribution --session-id <...> <contribution-path>`: sends a previously generated contribution file to the sequencer.

You might not need `kzgcli offline download-state` you're pulling the current state out-of-band (e.g: direct download or the sequencer sent it to you). If that isn't the case, you can use it in an environment that has internet access (not necessarily your contribution environment).

The `kzgcli offline contribute` command doesn't require internet access, and will probably be the only command you'll run in your constrained environment. This command also accepts the `--urlrand` flag if you want to pull entropy from an external source of randomness available in your environment.

The `kzgcli offline send-contribution` command sends the previously generated file by `kzgcli offline contribute` to the sequencer.

An example of running the first two commands:
```
$ kzgcli offline download-state current.json
Downloading current state... OK
Encoding and saving to current.json... OK
Saved current state in current.json
$ kzgcli offline contribute current.json new.json
Opening and parsing offline current state file...OK
Calculating contribution... OK
Success, saved contribution in new.json
```

## Verify the current sequencer transcript ourselves
The sequencer has [an API that provides a full transcript](https://seq.ceremony.ethereum.org/info/current_state) of all the contributions, so anyone can double-check the calculations to see if the result matches all the received contributions.

Having clients double-check sequencer calculations avoids having to trust that the sequencer is in the latest powers of Tau calculation.

To verify the current transcript:
```
$ kzgcli verify-transcript
Pulling current transcript from sequencer... OK
Verifying transcript... Valid! (took 13.08s)
```
Note that you don't need a `--session-id`, so anyone can run the verifying logic.

## Tests and benchmarks
You can run the tests for the repo doing `make test` or `go test ./... -race`.

You can run benchmarks with `make bench` or `go test ./... -run=none -bench=.`:
```
goos: linux
goarch: amd64
pkg: github.com/jsign/go-kzg-ceremony-client/contribution
cpu: AMD Ryzen 7 3800XT 8-Core Processor            
BenchmarkDecodeJSON-16                 1        2341867398 ns/op
BenchmarkContribute-16                 1        2964969708 ns/op
```

As shown, in a modern desktop CPU the contribution calculation takes less than 3 seconds. The only "optimization" done in the client is leveraging multiple cores to calculate your contribution. If your CPU is ~modern, gnark-crypto library might leverage special CPU instructions such as [ADX](https://en.wikipedia.org/wiki/Intel_ADX) to do some elliptic curve operations way faster (no configuration needed).

## Side-effects of this ceremony client work
While creating this ceremony client, I contributed to other repositories in the ecosystem:
- To validate this client implementation without a sequencer, I created the [kzg-ceremony-test-vectors](https://github.com/jsign/kzg-ceremony-test-vectors) repository which generates batch contributions from the spec initialContribution.json file with a fixed set of secrets producing a deterministic/reproducible output that clients can check against the sequencer reference implementation. [You can see the unit-test leveraging this test vector](https://github.com/jsign/go-kzg-ceremony-client/blob/917d4b5da6a54da4879fd8869e84344dd57ad950/contribution/contribution_test.go#L33).
- I detected a slight bug in one of the Rust clients and [fixed it](https://github.com/crate-crypto/small-powers-of-tau/pull/4).
- While trying to add ECDSA EIP-721 signature verification for the transcript, I found [an inconsistency](https://hackmd.io/@jsign/kzg-ceremony-eip712-problem) in how `eth-rs` or `go-ethereum` implement the EIP. This potential bug doesn't allow this client to verify ECDSA signatures in the transcript. ~~This situation is under investigation.~~ ([fixed in `go-ethereum` PR](https://github.com/ethereum/go-ethereum/pull/26462))


## Potential improvements
Despite this client is ready to contribute to the ceremony, there're a couple of things that it doesn't support but could if I can convince myself of some tradeoffs:
- The `gnark-crypto` library [doesn't support BLS signing yet](https://github.com/ConsenSys/gnark-crypto/issues/116), which is incredibly unfortunate. This doesn't allow the client to do BLS signing or verification in the transcript. This is an optional feature for clients, so it isn't a big deal or create any risk. Despite gnark-crypto has support for group multiplication and pairings, the biggest pending work for signing is implementing the _hash to curve_ step of signing which isn't entirely trivial. As a workaround, I could use a separate BLS library to do signing/verification, but that would mean using two BLS libraries and I'd prefer to be 100% clear about which library is used in this repo for _all_ cryptographic operations just by looking at the [go.mod](go.mod) file.

## License
MIT
