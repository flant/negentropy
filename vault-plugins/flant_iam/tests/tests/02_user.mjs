import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genTenantPayload, TenantEndpointBuilder } from "./lib/tenant.mjs"
import { expect } from "chai"
import { genUserPayload, SubTenantEntrypointBuilder } from "./lib/subtenant.mjs"
import { API } from "./lib/api.mjs"

//    /tenant/{tid}/user/{uid}

describe("User", function () {
    const rootClient = getClient(rootToken)
    const rootTenantAPI = new API(rootClient, new TenantEndpointBuilder())

    const entrypointBuilder = new SubTenantEntrypointBuilder("user")
    const rootUserClient = new API(rootClient, entrypointBuilder)

    function genPayload() {
        return genUserPayload()
    }

    async function createTenantId() {
        const payload = genTenantPayload()
        const { data } = await rootTenantAPI.create({ payload })
        return data.data.uuid
    }

    async function createSubtenantId(tid) {
        const payload = genPayload()
        const { data: body } = await rootUserClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data.uuid
    }

    it("can be created", async () => {
        const tid = await createTenantId()

        const { data: body } = await rootUserClient.create({
            params: { tenant: tid },
            payload: genPayload(),
        })

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.have.key("uuid")
        expect(body.data.uuid).to.be.a("string").of.length.above(10)
    })

    it("can be read", async () => {
        const tid = await createTenantId()

        // create
        const payload = genPayload()
        const { data: body } = await rootUserClient.create({
            params: { tenant: tid },
            payload,
        })
        const id = body.data.uuid

        // read
        const { data: user } = await rootUserClient.read({
            params: { tenant: tid, user: id },
        })
        expect(user.data).to.deep.eq({ uuid: id, tenant_uuid: tid })
    })

    it("can be updated", async () => {
        const tid = await createTenantId()

        // create
        const uid = await createSubtenantId(tid)

        // update
        const payload = genPayload()
        const params = { tenant: tid, user: uid }
        await rootUserClient.update({ params, payload })

        // read
        const { data: body } = await rootUserClient.read({ params })
        const sub = body.data

        expect(sub).to.deep.eq({ uuid: uid, tenant_uuid: tid })
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const uid = await createSubtenantId(tid)

        // delete
        const params = { tenant: tid, user: uid }
        await rootUserClient.delete({ params })

        // read
        await rootUserClient.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const uid = await createSubtenantId(tid)

        // delete
        const params = { tenant: tid }
        const { data: body } = await rootUserClient.list({ params })

        expect(body.data).to.be.an("object").and.include.keys("uuids")
        expect(body.data.uuids).to.be.an("array").of.length(1) // if not 1, maybe users are not filtered by tenants
        expect(body.data.uuids[0]).to.eq(uid)
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        const params = { user: "no-such" }

        before("create tenant", async () => {
            params.tenant = await createTenantId()
        })

        it("cannot read, gets 404", async () => {
            await rootUserClient.read({ params, opts })
        })

        it("cannot update, gets 404", async () => {
            await rootUserClient.update({
                params,
                opts,
                payload: genTenantPayload(),
            })
        })

        it("cannot delete, gets 404", async () => {
            await rootUserClient.delete({ params, opts })
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
            const opts = expectStatus(expectedStatus)
            const unauth = new API(client, entrypointBuilder)
            let payload = {}

            const params = {}
            before("create tenant", async () => {
                params.tenant = await createTenantId()
            })

            beforeEach("create user payload", async () => {
                payload = genPayload()
            })

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ params, payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ params, opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const { data } = await rootUserClient.create({
                    params,
                    payload,
                })
                const uid = data.data.uuid

                await unauth.read({
                    params: { ...params, user: uid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const { data } = await rootUserClient.create({
                    params,
                    payload,
                })
                const uid = data.data.uuid
                await unauth.update({
                    params: { ...params, user: uid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const { data } = await rootUserClient.create({
                    params,
                    payload,
                })
                const uid = data.data.uuid
                await unauth.delete({
                    params: { ...params, user: uid },
                    opts,
                })
            })
        }
    })
})
