import _ from "lodash"
import { expectStatus } from "./client.mjs"
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

export class API {
    constructor(client, endpointBuilder, responseMapper) {
        this.client = new CRUD(client)
        this.endpointBuilder = endpointBuilder
        this.responseMapper = responseMapper || new NoopResponseMapper()
    }

    async create({ params = {}, query = {}, payload, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.collection(params, query)
        const { data: body } = await this.client.post(endpoint, payload, {
            ...expectStatus(201),
            ...opts,
        })
        return this.responseMapper.mapOne(body)
    }

    async createPrivileged({ params = {}, query = {}, payload, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.privileged(params, query)
        const { data: body } = await this.client.post(endpoint, payload, {
            ...expectStatus(201),
            ...opts,
        })
        return this.responseMapper.mapOne(body)
    }

    async read({ params = {}, query = {}, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.one(params, query)
        const { data: body } = await this.client.get(endpoint, {
            ...expectStatus(200),
            ...opts,
        })
        return this.responseMapper.mapOne(body)
    }

    async update({ params = {}, query = {}, payload, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.one(params, query)
        const { data: body } = await this.client.post(endpoint, payload, {
            ...expectStatus(200),
            ...opts,
        })
        return this.responseMapper.mapOne(body)
    }

    async delete({ params = {}, query = {}, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.one(params, query)
        const { data: body } = await this.client.delete(endpoint, {
            ...expectStatus(204),
            ...opts,
        })
        return this.responseMapper.mapOne(body)
    }

    async list({ params = {}, query = { list: true }, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.collection(params, query)
        const { data: body } = await this.client.get(endpoint, {
            ...expectStatus(200),
            ...opts,
        })
        return this.responseMapper.mapMany(body)
    }
}

class NoopResponseMapper {
    mapOne(x) {
        return x
    }
    mapMany(x) {
        return x
    }
}

export class SingleFieldReponseMapper {
    constructor(onePath = "data", listPath = "data") {
        this.onePath = onePath
        this.listPath = listPath
    }

    mapOne(x) {
        return _.get(x, this.onePath)
    }

    mapMany(x) {
        return _.get(x, this.listPath)
    }
}
