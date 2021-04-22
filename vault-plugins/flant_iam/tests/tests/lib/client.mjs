import A from "axios"

import fs from "fs"

import path from "path"

const baseUrl = "http://127.0.0.1:8200/v1/"
const pluginPath = "flant_iam"

function getSecondRootToken() {
    const token = fs.readFileSync(path.resolve("./data/token"))
    return token.toString().trim()
}

export const rootToken = "root"
export const anotherToken = getSecondRootToken()

export function expectStatus(expectedStatus) {
    return {
        validateStatus: (x) => x === expectedStatus,
    }
}

export function getClient(token) {
    const baseURL = baseUrl + pluginPath

    const headers = {
        "Content-Type": "application/json",
        Accept: "application/json",
    }

    if (token) {
        headers["X-Vault-Token"] = token
    }

    const client = A.create({ baseURL, headers })
    client.interceptors.response.use(null, axiosErrFormatter)
    return client
}

/**
 * axiosErrFormatter lets us read prettified response errors. Used in the axios error interception reponse hook.
 *
 * @param {Error} err
 *
 * @example ```
 *      client.interceptors.response.use(null, axiosErrFormatter)
 *  ```
 */
function axiosErrFormatter(err) {
    // Log and throw further
    const sent = err.request.method + " " + err.request.path
    const status = `STATUS: ${err.response.status}`
    const body = err.response.data
        ? JSON.stringify(err.response.data, null, 2)
        : ""

    const prefixize = (pad, text) =>
        text
            .split("\n")
            .map((s) => pad + s)
            .join("\n")

    const msg = [
        "\n",
        prefixize("     →  ", sent),
        "",
        prefixize("     ←  ", [status, body].join("\n")),
    ].join("\n")

    // console.error(msg)
    err.message += msg
    throw err
}
