import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genTenantPayload, TenantEndpointBuilder } from "./lib/tenant.mjs"
import { expect } from "chai"
import {
    EndpointBuilder,
    genMultipassPayload,
    genUserPayload,
    SubTenantEntrypointBuilder,
} from "./lib/subtenant.mjs"
import { API } from "./lib/api.mjs"

//    /tenant/{tid}/user/{uid}

describe("User", function () {
    const rootClient = getClient(rootToken)
    const rootTenantAPI = new API(rootClient, new TenantEndpointBuilder())

    const entrypointBuilder = new SubTenantEntrypointBuilder("user")
    const rootUserClient = new API(rootClient, entrypointBuilder)

    function genPayload(override) {
        return genUserPayload(override)
    }

    async function createTenant() {
        const payload = genTenantPayload()
        const { data } = await rootTenantAPI.create({ payload })
        return data.data
    }

    async function createTenantId() {
        const tenant = await createTenant()
        return tenant.uuid
    }

    async function createUser(tid) {
        const payload = genPayload()
        const { data: body } = await rootUserClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data
    }

    it("can be created", async () => {
        const tid = await createTenantId()

        const { data: body } = await rootUserClient.create({
            params: { tenant: tid },
            payload: genPayload(),
        })

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.include.keys(
            "uuid",
            "tenant_uuid",
            "resource_version",
        )
        expect(body.data.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(body.data.tenant_uuid).to.eq(tid)
        expect(body.data.resource_version)
            .to.be.a("string")
            .of.length.greaterThan(5)
    })

    it("can be read", async () => {
        const tenant = await createTenant()
        const tid = tenant.uuid
        // create
        const payload = genPayload()
        const { data: created } = await rootUserClient.create({
            params: { tenant: tid },
            payload,
        })
        const uid = created.data.uuid
        const generated = {
            // email: "",
            uuid: created.data.uuid,
            tenant_uuid: created.data.tenant_uuid,
            resource_version: created.data.resource_version,
            full_identifier: payload.identifier + "@" + tenant.identifier,
        }

        // read
        const { data: read } = await rootUserClient.read({
            params: { tenant: tid, user: uid },
        })

        expect(read.data).to.deep.eq(
            { ...payload, ...generated },
            "must have generated fields",
        )
        expect(read.data).to.deep.eq(
            created.data,
            "reading and creation responses should contain the same data",
        )
        expect(read.data.resource_version)
            .to.be.a("string")
            .of.length.greaterThan(5)
    })

    it("can be updated", async () => {
        const tid = await createTenantId()

        // create
        const created = await createUser(tid)

        // update
        const payload = genPayload({
            resource_version: created.resource_version,
        })
        const params = { tenant: tid, user: created.uuid }
        const { data: updated } = await rootUserClient.update({
            params,
            payload,
        })

        // read
        const { data: read } = await rootUserClient.read({ params })
        const sub = read.data

        expect(read.data).to.deep.eq(updated.data)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const user = await createUser(tid)
        const uid = user.uuid

        // delete
        const params = { tenant: tid, user: uid }
        await rootUserClient.delete({ params })

        // read
        await rootUserClient.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const user = await createUser(tid)
        const uid = user.uuid

        // delete
        const params = { tenant: tid }
        const { data: body } = await rootUserClient.list({ params })

        expect(body.data).to.be.an("object").and.include.keys("uuids")
        expect(body.data.uuids).to.be.an("array").of.length(1) // if not 1, maybe users are not filtered by tenants
        expect(body.data.uuids[0]).to.eq(uid)
    })

    it("can be deleted by the tenant deletion", async () => {
        const tid = await createTenantId()
        const user = await createUser(tid)

        await rootTenantAPI.delete({ params: { tenant: tid } })

        const params = { tenant: tid, user: user.uuid }
        const opts = expectStatus(404)
        await rootUserClient.read({ params, opts })
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

    describe("multipass", function () {
        const endpointBuilder = new EndpointBuilder([
            "tenant",
            "user",
            "multipass",
        ])
        const rootMPClient = new API(rootClient, endpointBuilder)

        async function createMultipass(t, u, override = {}) {
            const payload = genMultipassPayload(override)
            const params = { tenant: t, user: u }
            const { data } = await rootMPClient.create({ params, payload })
            return data.data
        }

        it("can be created", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)

            await createMultipass(t.uuid, u.uuid)
        })

        it("contains expected data when created", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)

            const mp = await createMultipass(t.uuid, u.uuid)

            expect(mp)
                .to.be.an("object")
                .and.have.all.keys(
                    "allowed_cidrs",
                    "allowed_roles",
                    "description",
                    "max_ttl",
                    "owner_type",
                    "owner_uuid",
                    "tenant_uuid",
                    "salt",
                    "ttl",
                    "uuid",
                    "valid_till",
                )

            expect(mp.uuid, "uuid").to.be.a("string")
            expect(mp.owner_type, "owner_type").to.be.a("string")
            expect(mp.owner_uuid, "owner_uuid").to.be.a("string")
            expect(mp.tenant_uuid, "tenant_uuid").to.be.a("string")

            expect(mp.salt, "salt").to.be.a("string").and.to.be.empty // FIXME omit from the response

            expect(mp.description, "description").to.be.a("string")

            expect(mp.allowed_cidrs, "allowed_cidrs").to.be.an("array")
            expect(mp.allowed_roles, "allowed_roles").to.be.an("array")

            expect(mp.ttl, "ttl").to.be.a("number")
            expect(mp.max_ttl, "max_ttl").to.be.a("number")
            expect(mp.valid_till, "valid_till")
                .to.be.a("number")
                .greaterThan(Date.now() / 1e3)
        })

        it("can be read", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)
            const created = await createMultipass(t.uuid, u.uuid)

            const params = {
                tenant: t.uuid,
                user: u.uuid,
                multipass: created.uuid,
            }
            const { data } = await rootMPClient.read({ params })
            const read = data.data

            expect(read).to.deep.eq(created)
        })

        it("can be listed", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)

            const createId = () =>
                createMultipass(t.uuid, u.uuid).then((mp) => mp.uuid)

            const ids = await Promise.all([createId(), createId(), createId()])

            const params = {
                tenant: t.uuid,
                user: u.uuid,
            }

            const { data } = await rootMPClient.list({ params })
            console.log(data)

            expect(data.data.ids).to.have.all.members(...ids)
        })

        it("can be deleted", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)
            const created = await createMultipass(t.uuid, u.uuid)

            const params = {
                tenant: t.uuid,
                user: u.uuid,
                multipass: created.uuid,
            }

            await rootMPClient.delete({ params })

            await rootMPClient.read({ params, opts: expectStatus(404) })
        })

        it("cannot be updated", async () => {
            const t = await createTenant()
            const u = await createUser(t.uuid)
            const createdMP = await createMultipass(t.uuid, u.uuid)

            const params = {
                tenant: t.uuid,
                user: u.uuid,
                multipass: createdMP.uuid,
            }

            const { data } = await rootMPClient.update({
                params,
                payload: genMultipassPayload(),
                opts: expectStatus(405),
            })
        })
    })
})
