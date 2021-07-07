import { expectStatus } from "./client.mjs"
import { join } from "path"

class CRUD {
    constructor(client) {
        this.client = client
    }

    get(endpoint, opts) {
        return this.client.get(endpoint, opts)
    }

    post(endpoint, payload, opts) {
        return this.client.post(endpoint, payload, opts)
    }

    put(endpoint, payload, opts) {
        return this.client.put(endpoint, payload, opts)
    }

    delete(endpoint, opts) {
        return this.client.delete(endpoint, opts)
    }
}

export function stringifyQuery(q = {}) {
    if (!q || Object.keys(q).length === 0) {
        return ""
    }
    return "?" + new URLSearchParams(q).toString()
}

// Example class, not to be used
export class ExampleEndpointBuilder {
    constructor() {
        this.prefix = "items"
    }

    one(p = {}, q = {}) {
        return join(this.prefix, p.item) + stringifyQuery(q)
    }

    collection(p = {}, q = {}) {
        return this.prefix + stringifyQuery(q)
    }

    privileged(p = {}, q = {}) {
        return join(this.prefix, "privileged") + stringifyQuery(q)
    }
}

export class API {
    constructor(client, endpointBuilder) {
        this.client = new CRUD(client)
        this.endpointBuilder = endpointBuilder
    }

    create({ params = {}, query = {}, payload, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.collection(params, query)
        return this.client.post(endpoint, payload, {
            ...expectStatus(201),
            ...opts,
        })
    }

    createPrivileged({ params = {}, query = {}, payload, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.privileged(params, query)
        return this.client.post(endpoint, payload, {
            ...expectStatus(201),
            ...opts,
        })
    }

    read({ params = {}, query = {}, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.one(params, query)
        return this.client.get(endpoint, {
            ...expectStatus(200),
            ...opts,
        })
    }

    update({ params = {}, query = {}, payload, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.one(params, query)
        return this.client.post(endpoint, payload, {
            ...expectStatus(200),
            ...opts,
        })
    }

    delete({ params = {}, query = {}, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.one(params, query)
        return this.client.delete(endpoint, {
            ...expectStatus(204),
            ...opts,
        })
    }

    list({ params = {}, query = { list: true }, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.collection(params, query)
        return this.client.get(endpoint, {
            ...expectStatus(200),
            ...opts,
        })
    }
}
