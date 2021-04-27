import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genTenantPayload, TenantAPI } from "./lib/tenant.mjs"
import { expect } from "chai"
import { v4 as uuidv4 } from "uuid"

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
        describe("identifier", () => {
            const invalidCases = [
                {
                    title: "number allowed", // the matter of fact ¯\_(ツ)_/¯
                    payload: genTenantPayload({ identifier: 100 }),
                    validateStatus: (x) => x === 201,
                },
                {
                    title: "absent identifier forbidden",
                    payload: (() => {
                        const p = genTenantPayload({})
                        delete p.identifier
                        return p
                    })(),
                    validateStatus: (x) => x === 400,
                },
                {
                    title: "empty string forbidden",
                    payload: genTenantPayload({ identifier: "" }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "array forbidden",
                    payload: genTenantPayload({ identifier: ["a"] }),
                    validateStatus: (x) => x >= 400, // 500 is allowed
                },
                {
                    title: "object forbidden",
                    payload: genTenantPayload({ identifier: { a: 1 } }),
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
        expect(body.data).to.include.keys(
            "uuid",
            "identifier",
            "resource_version",
        )
        expect(body.data.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(body.data.resource_version)
            .to.be.a("string")
            .of.length.greaterThan(5)
    })

    it("can be read", async () => {
        const payload = genTenantPayload()

        const { data: created } = await root.create(payload)
        const { data: read } = await root.read(created.data.uuid)

        expect(read.data).to.deep.eq(created.data)
        expect(created.data).to.deep.contain({
            ...payload,
            uuid: created.data.uuid,
        })
        expect(created.data.resource_version)
            .to.be.a("string")
            .of.length.greaterThan(5)
    })

    it("can be read by id", async () => {
        const payload1 = genTenantPayload()
        const payload2 = genTenantPayload()
        const payload3 = genTenantPayload()

        const { data: body1 } = await root.create(payload1)
        const id1 = body1.data.uuid
        const { data: body2 } = await root.create(payload2)
        const id2 = body2.data.uuid
        const { data: body3 } = await root.create(payload3)
        const id3 = body3.data.uuid

        const { data: resp1 } = await root.read(id1)
        const { data: resp2 } = await root.read(id2)
        const { data: resp3 } = await root.read(id3)

        expect(resp1.data).to.contain({ ...payload1, uuid: id1 })
        expect(resp2.data).to.contain({ ...payload2, uuid: id2 })
        expect(resp3.data).to.contain({ ...payload3, uuid: id3 })
    })

    it("can be updated", async () => {
        const createPld = genTenantPayload()
        const updatePld = genTenantPayload()

        // create
        const { data: body1 } = await root.create(createPld)
        const uuid = body1.data.uuid
        const resource_version = body1.data.resource_version

        // update
        const { data: body2 } = await root.update(uuid, {
            ...updatePld,
            resource_version,
        })

        // read
        const { data: body3 } = await root.read(uuid)
        const tenant = body3.data

        expect(tenant).to.contain(
            { ...updatePld, uuid },
            "payload must be saved",
        )
        expect(tenant.resource_version)
            .to.be.a("string")
            .of.length.greaterThan(5)
            .and.not.to.eq(resource_version, "resource version must be updated")
    })

    it("can be deleted", async () => {
        const createPld = genTenantPayload()

        const { data: body1 } = await root.create(createPld)
        const id = body1.data.uuid

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
        const id = creationBody.data.uuid

        const { data: listBody } = await root.list()

        expect(listBody.data).to.be.an("object").and.have.key("uuids")
        expect(listBody.data.uuids).to.include(id)
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

    describe("no access", function () {
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
                await unauth.read(data.data.uuid, opts)
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const { data } = await root.create(genTenantPayload())
                await unauth.update(data.data.uuid, genTenantPayload(), opts)
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const { data } = await root.create(genTenantPayload())
                await unauth.delete(data.data.uuid, opts)
            })
        }
    })

    describe("privileged access", function () {
        it(`creates`, async () => {
            const payload = genTenantPayload({ uuid: uuidv4() })

            const { data: body } = await root.createPriveleged(payload)

            const id = body.data.uuid
            expect(id).to.deep.eq(payload.uuid)
        })
    })
})
