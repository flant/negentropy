import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import {
    genTenantPayload,
    TenantAPI,
    TenantEndpointBuilder,
} from "./lib/tenant.mjs"
import { expect } from "chai"
import { API } from "./lib/api.mjs"

describe("Tenants", function () {
    const rootClient = getClient(rootToken)
    const root = new TenantAPI(rootClient)

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
