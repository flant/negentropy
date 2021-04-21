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

class Tenants {
    constructor(client) {
        this.client = client
    }

    create(payload, opts = {}) {
        return this.client.post("/tenant", payload, {
            ...expectStatus(201),
            ...opts,
        })
    }

    read(id, opts = {}) {
        return this.client.get(`/tenant/${id}`, {
            ...expectStatus(200),
            ...opts,
        })
    }

    update(id, payload, opts = {}) {
        return this.client.post(`/tenant/${id}`, payload, {
            ...expectStatus(204),
            ...opts,
        })
    }

    delete(id, opts = {}) {
        return this.client.delete(`/tenant/${id}`, {
            ...expectStatus(204),
            ...opts,
        })
    }

    list(opts = {}) {
        return this.client.get("/tenant?list=true", {
            ...expectStatus(200),
            ...opts,
        })
    }
}

describe("Tenants", function () {
    const rootClient = getClient(rootToken)
    const root = new Tenants(rootClient)
    const worder = new Worder()

    // afterEach("cleanup", async function () {
    //     const clean = s => root.delete(`tenant/${s}`, expectStatus(204))
    //     const promises = worder.list().map(clean)
    //     await Promise.all(promises)
    //     worder.clean()
    // })

    it("responds with 404 on inexisting", async () => {
        await root.read("no-such", { validateStatus: (s) => s === 404 })
    })

    it("cannot be created without name", async () => {
        await root.create({}, expectStatus(400))
    })

    it("cannot be created with empty name", async () => {
        await root.create({ name: "" }, expectStatus(400))
    })

    it("can be created with a name", async () => {
        const payload = { name: worder.gen() }

        const { data: body } = await root.create(payload)

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.have.key("id")
        expect(body.data.id).to.be.a("string").of.length.above(10)
    })

    it("can be read", async () => {
        const payload = { name: worder.gen() }

        const { data: body } = await root.create(payload)
        const id = body.data.id

        const { data: tenant } = await root.read(id)
        expect(tenant.data).to.include.keys("name")
        expect(tenant.data.name).to.eq(payload.name)
    })

    it("can be listed", async () => {
        const payload = { name: worder.gen() }
        await root.create(payload, expectStatus(204))

        const { data } = await root.list(expectStatus(200))

        expect(data.data).to.be("array").and.to.be.not.empty
    })

    it("has identifying fields in list", async () => {
        const payload = { name: worder.gen() }
        await root.create(payload, expectStatus(204))

        const { data: body } = await root.list()

        expect(body.data).to.be("array").and.to.be.not.empty
        const tenant = body.data[0]
        expect(tenant).to.include.keys("id", "name")
        expect(tenant.name).to.eq(payload.name)
    })
})
