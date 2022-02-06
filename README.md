# Backgammon Web API

Backgammon Web API. Sophisticed neural net based multi-ply evalution engine for Backgammon moves.

Based on GNU Backgammon (https://www.gnu.org/software/gnubg) under GPL license.

Features:

- Calculate best moves for a given Backgammon position

Features to-do:

- Calculate cube decisions

---

**Want to see the Backgammon Web API in action?** Have a look at https://github.com/foochu/bgweb-terminal.

---

## Running the REST API server

### Run via Docker

```sh
# 1 - install docker

# 2 - run the program:
docker run -p 8080:8080 -d foochu/bgweb-api:latest

# 3 - browse to http://localhost:8080
```

### Run from source

```sh
# 1 - install Go

# 2 - clone this repo

# 3 - run the program:
go run ./cmd/bgweb-api

# 4 - browse to http://localhost:8080
```

## Get best moves

### Parameters

- `board` = Board layout
  - `x` = Layout for player `x`
    - `1` - `24` = Number of chequers at each point
    - `bar` = Number of chequers on bar
  - `o` = Layout for player `o`
    - `1` - `24` = Number of chequers at each point
    - `bar` = Number of chequers on bar
- `cubeful` = Is doubling cube at play? Affects equity algorithm.
- `dice` = 2-slot array of dice roll
- `max-moves` = Max number of moves to return
- `player` = Player who's turn it is to move, either `x` or `o`
- `score-moves` = Calculate equity & winning chance. If `false` just returns list of legal moves.

### Example

For example, get top moves for starting position and dice roll 3-1 for player `x`:

```
curl -L -X POST 'http://localhost:8080/api/v1/getmoves' \
-H 'accept: application/json' \
-H 'Content-Type: application/json' \
--data-raw '{
  "board": {
    "o": {
      "6": 5,
      "8": 3,
      "13": 5,
      "24": 2
    },
    "x": {
      "6": 5,
      "8": 3,
      "13": 5,
      "24": 2
    }
  },
  "cubeful": false,
  "dice": [3, 1],
  "max-moves": 3,
  "player": "x",
  "score-moves": true
}'
```

Return moves in order of preference based on equity and winning chance:

```json
[
  {
    "play": [
      {
        "from": "8",
        "to": "5"
      },
      {
        "from": "6",
        "to": "5"
      }
    ],
    "evaluation": {
      "info": {
        "cubeful": false,
        "plies": 1
      },
      "eq": 0.159,
      "diff": 0,
      "probability": {
        "win": 0.551,
        "winG": 0.174,
        "winBG": 0.013,
        "lose": 0.449,
        "loseG": 0.124,
        "loseBG": 0.005
      }
    }
  },
  {
    "play": [
      {
        "from": "13",
        "to": "10"
      },
      {
        "from": "24",
        "to": "23"
      }
    ],
    "evaluation": {
      "info": {
        "cubeful": false,
        "plies": 1
      },
      "eq": -0.009,
      "diff": -0.168,
      "probability": {
        "win": 0.497,
        "winG": 0.137,
        "winBG": 0.008,
        "lose": 0.503,
        "loseG": 0.14,
        "loseBG": 0.007
      }
    }
  },
  {
    "play": [
      {
        "from": "24",
        "to": "21"
      },
      {
        "from": "21",
        "to": "20"
      }
    ],
    "evaluation": {
      "info": {
        "cubeful": false,
        "plies": 1
      },
      "eq": -0.015,
      "diff": -0.175,
      "probability": {
        "win": 0.497,
        "winG": 0.125,
        "winBG": 0.005,
        "lose": 0.503,
        "loseG": 0.135,
        "loseBG": 0.004
      }
    }
  }
]
```

## Web Assembly

Web Assembly allows to run the API functions directly in the browser without a need for backend server. Logic, runtime & data files are all bundled into a single file.

Build wasm:

```sh
# 1 - install Go

# 2 - clone this repo

# 3 - build wasm
./scripts/buildwasm.sh

# 4 - generates `lib.wasm`
```

In your web app:

```js
const go = new Go();

WebAssembly.instantiateStreaming(fetch("lib.wasm"), go.importObject).then(
  async (result) => {
    await go.run(result.instance);
  }
);
```

Web Assembly declares global JS function `wasm_get_moves()`. Example usage:

```ts
let input = JSON.stringify({
  board: {
    o: {
      "6": 5,
      "8": 3,
      "13": 5,
      "24": 2,
    },
    x: {
      "6": 5,
      "8": 3,
      "13": 5,
      "24": 2,
    },
  },
  cubeful: false,
  dice: [3, 1],
  "max-moves": 3,
  player: "x",
  "score-moves": true,
});

let output = global.wasm_get_moves(input);

let moves = JSON.parse(output);

console.log(moves);
```
