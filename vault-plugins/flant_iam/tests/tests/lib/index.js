const A = require("axios")
const F = require("faker")
const fs = require("fs")
const path = require("path")

const baseUrl = "http://127.0.0.1:8200/v1/"
const pluginPath = "flant_iam"

function getClient(token) {
    const instance = A.create({
        baseURL: baseUrl + pluginPath,
        headers: {
            "X-Vault-Token": token,
            "Content-Type": "application/json",
            Accept: "application/json",
        },
    })

    instance.interceptors.response.use(null, (err) => {
        // Log and throw further
        const sent = err.request.method + " " + err.request.path
        const status = `STATUS: ${err.response.status}`
        const body = err.response.data
            ? JSON.stringify(err.response.data, null, 2)
            : ""

        const prefixize = (pad, text) => text.split("\n").map(s => pad + s).join("\n")

        const msg = [
            "\n",
            prefixize("     →  ", sent),
            "",
            prefixize("     ←  ", [status, body].join("\n")),
        ].join("\n")

        // console.error(msg)
        err.message += msg
        throw err
    })

    return instance
}

function expectStatus(expectedStatus) {
    return {
        validateStatus: (x) => x === expectedStatus,
    }
}

class Worder {
    constructor() {
        this.set = new Set()
    }

    gen(prefix = "") {
        const word = F.lorem.word()
        this.set.add(word)
        // console.log("Worder gen", word)
        return word
    }

    list() {
        // console.log("Worder list", Array.from(this.set))
        return Array.from(this.set)
    }

    clean() {
        // console.log("Worder clean")
        this.set = new Set()
    }
}

function getSecondRootToken() {
    const token = fs.readFileSync(path.resolve("./data/token"))
    return token.toString().trim()
}

module.exports = {
    getClient,
    Worder,
    expectStatus,
    anotherToken: getSecondRootToken(),
    rootToken: "root",
}
