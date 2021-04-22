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
            ...expectStatus(200),
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

    it("can be updated", async () => {
        const createPld = { name: worder.gen() }
        const updatePld = { name: worder.gen() }

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
        const createPld = { name: worder.gen() }

        const { data: body1 } = await root.create(createPld)
        const id = body1.data.id

        await root.delete(id)

        await root.read(id, expectStatus(404))
    })

    it("can be listed", async () => {
        const payload = { name: worder.gen() }
        await root.create(payload)

        const { data } = await root.list()

        expect(data.data).to.be.an("object")
    })

    it("has identifying fields in list", async () => {
        const payload = { name: worder.gen() }
        const { data: creationBody } = await root.create(payload)
        const id = creationBody.data.id

        const { data: listBody } = await root.list()

        expect(listBody.data).to.be.an("object").and.have.key("ids")
        const { ids } = listBody.data
        expect(ids).to.include(id)
    })
})
