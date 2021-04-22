const { expect } = require("chai")
const {
    Worder,
    expectStatus,
    getClient,
    rootToken,
    anotherToken,
} = require("./lib")

const TEN = "tenant"
const tenId = (id) => TEN + "/" + id

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

    delete(endpoint, opts) {
        return this.client.delete(endpoint, opts)
    }
}

class API {
    constructor(client, endpointBuilder) {
        this.client = new CRUD(client)
        this.endpointBuilder = endpointBuilder
    }

    create({ params = {}, query = {}, payload, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.create(params, query)
        return this.client.post(endpoint, payload, {
            ...expectStatus(201),
            ...opts,
        })
    }

    read({ params = {}, query = {}, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.read(params, query)
        return this.client.get(endpoint, {
            ...expectStatus(200),
            ...opts,
        })
    }

    update({ params = {}, query = {}, payload, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.update(params, query)
        return this.client.post(endpoint, payload, {
            ...expectStatus(200),
            ...opts,
        })
    }

    delete({ params = {}, query = {}, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.delete(params, query)
        return this.client.delete(endpoint, {
            ...expectStatus(204),
            ...opts,
        })
    }

    list({ params = {}, query = {}, opts = {} } = {}) {
        const endpoint = this.endpointBuilder.list(params, query)
        return this.client.get(endpoint, {
            ...expectStatus(200),
            ...opts,
        })
    }
}

class TenantEndpointBuilder {
    create(p = {}, q = {}) {
        return "/tenant"
    }

    read(p = {}, q = {}) {
        return `/tenant/${p.tenant}`
    }

    update(p = {}, q = {}) {
        return `/tenant/${p.tenant}`
    }

    delete(p = {}, q = {}) {
        return `/tenant/${p.tenant}`
    }

    list(p = {}, q = {}) {
        return "/tenant?list=true"
    }
}

class TenantAPI {
    constructor(client) {
        this.api = new API(client, new TenantEndpointBuilder())
    }

    create(payload, opts) {
        return this.api.create({ payload, opts })
    }

    read(id, opts) {
        return this.api.read({ params: { tenant: id }, opts })
    }

    update(id, payload, opts) {
        return this.api.update({ params: { tenant: id }, payload, opts })
    }

    delete(id, opts) {
        return this.api.delete({ params: { tenant: id }, opts })
    }

    list(opts) {
        return this.api.list({ opts })
    }
}

describe("Tenants", function () {
    const rootClient = getClient(rootToken)
    const root = new TenantAPI(rootClient)
    const worder = new Worder()

    function genTenantPayload(override = {}) {
        return {
            name: worder.gen(),
            ...override,
        }
    }

    // afterEach("cleanup", async function () {
    //     const clean = s => root.delete(`tenant/${s}`, expectStatus(204))
    //     const promises = worder.list().map(clean)
    //     await Promise.all(promises)
    //     worder.clean()
    // })

    describe("payload", () => {
        describe("name", () => {
            const invalidCases = [
                {
                    title: "number allowed",
                    payload: genTenantPayload({ name: 0 }),
                    validateStatus: (x) => x === 201,
                },
                {
                    title: "absent name field forbidden",
                    payload: (() => {
                        const p = genTenantPayload({})
                        delete p.name
                        return p
                    })(),
                    validateStatus: (x) => x === 400,
                },
                {
                    title: "empty string forbidden",
                    payload: genTenantPayload({ name: "" }),
                    validateStatus: (x) => x === 400,
                },
                {
                    title: "array forbidden",
                    payload: genTenantPayload({ name: ["a"] }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "object forbidden",
                    payload: genTenantPayload({ name: { a: 1 } }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
            ]

            invalidCases.forEach((x) =>
                it(x.title, async () => {
                    await root.create(x.payload, {
                        validateStatus: x.validateStatus,
                    })
                }),
            )
        })
    })

    it("can be created", async () => {
        const payload = genTenantPayload()

        const { data: body } = await root.create(payload)

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.have.key("id")
        expect(body.data.id).to.be.a("string").of.length.above(10)
    })

    it("can be read", async () => {
        const payload = genTenantPayload()

        const { data: body } = await root.create(payload)
        const id = body.data.id

        const { data: tenant } = await root.read(id)
        expect(tenant.data).to.include.keys("name")
        expect(tenant.data.name).to.eq(payload.name)
    })

    it("responds with 404 on inexisting", async () => {
        await root.read("no-such", { validateStatus: (s) => s === 404 })
    })

    it("can be updated", async () => {
        const createPld = genTenantPayload()
        const updatePld = genTenantPayload()

        // create
        const { data: body1 } = await root.create(createPld)
        const id = body1.data.id

        // update
        const { data: body2 } = await root.update(id, updatePld)

        // read
        const { data: body3 } = await root.read(id)
        const tenant = body3.data

        expect(tenant).to.include.all.keys("name")
        expect(tenant.name).to.eq(updatePld.name)
    })

    it("can be deleted", async () => {
        const createPld = genTenantPayload()

        const { data: body1 } = await root.create(createPld)
        const id = body1.data.id

        await root.delete(id)

        await root.read(id, expectStatus(404))
    })

    it("can be listed", async () => {
        const payload = genTenantPayload()
        await root.create(payload)

        const { data } = await root.list()

        expect(data.data).to.be.an("object")
    })

    it("has identifying fields in list", async () => {
        const payload = genTenantPayload()
        const { data: creationBody } = await root.create(payload)
        const id = creationBody.data.id

        const { data: listBody } = await root.list()

        expect(listBody.data).to.be.an("object").and.have.key("ids")
        const { ids } = listBody.data
        expect(ids).to.include(id)
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        it("cannot read, gets 404", async () => {
            await root.read("no-such", opts)
        })

        it("cannot update, gets 404", async () => {
            await root.update("no-such", genTenantPayload(), opts)
        })

        it("cannot delete, gets 404", async () => {
            await root.delete("no-such", opts)
        })
    })

    describe("access", function () {
        describe("when unauthenticated", function () {
            runWithClient(getClient(), 400)
        })

        describe("when unauthorized", function () {
            runWithClient(getClient("xxx"), 403)
        })

        function runWithClient(client, expectedStatus) {
            const unauth = new TenantAPI(client)
            const opts = expectStatus(expectedStatus)

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create(genTenantPayload(), opts)
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list(opts)
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const { data } = await root.create(genTenantPayload())
                await unauth.read(data.data.id, opts)
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const { data } = await root.create(genTenantPayload())
                await unauth.update(data.data.id, genTenantPayload(), opts)
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const { data } = await root.create(genTenantPayload())
                await unauth.delete(data.data.id, opts)
            })
        }
    })
})
