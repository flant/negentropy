import { expect } from "chai"
import { API } from "./lib/api.mjs"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import {
    genGroupPayload,
    genUserPayload,
    genServiceAccountPayload,
    SubTenantEntrypointBuilder,
} from "./lib/subtenant.mjs"
import { genTenantPayload, TenantEndpointBuilder } from "./lib/tenant.mjs"

//    /tenant/{tid}/group/{gid}

describe("Group", function () {
    const rootClient = getClient(rootToken)
    const rootTenantAPI = new API(rootClient, new TenantEndpointBuilder())

    const entrypointBuilder = new SubTenantEntrypointBuilder("group")
    const rootGroupClient = new API(rootClient, entrypointBuilder)

    // Clients to provide dependencies
    const userEntrypointBuilder = new SubTenantEntrypointBuilder("user")
    const saEntrypointBuilder = new SubTenantEntrypointBuilder("service_account")
    const rootUserClient = new API(rootClient, userEntrypointBuilder)
    const rootServiceAccountClient = new API(rootClient, saEntrypointBuilder)

    function genPayload(override) {
        return genRoleBindingPayload(override)
    }

    async function createTenant() {
        const payload = genTenantPayload()
        const { data: body } = await rootTenantAPI.create({ payload })
        console.log("tenant response body", body)
        return body.data.tenant
    }

    async function createTenantId() {
        const tenant = await createTenant()
        return tenant.uuid
    }

    async function createServiceAccount(tid) {
        const payload = genServiceAccountPayload()
        const { data: body } = await rootServiceAccountClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data.service_account
    }

    async function createUser(tid) {
        const payload = genUserPayload()
        const { data: body } = await rootUserClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data.user
    }

    async function createSubjects(tid) {
        const user = await createUser(tid)
        const sa = await createServiceAccount(tid)
        return [subject("user", user.uuid), subject("service_account", sa.uuid)]
    }

    function subject(type, id) {
        return { type, id }
    }

    function genPayload(override) {
        return genGroupPayload(override)
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

    async function createGroup(tid) {
        const subjects = await createSubjects(tid)
        const payload = genPayload({ subjects })
        const { data: body } = await rootGroupClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data
    }

    it("can be created", async () => {
        const tid = await createTenantId()
        const subjects = await createSubjects(tid)

        const { data: body } = await rootGroupClient.create({
            params: { tenant: tid },
            payload: genPayload({ subjects }),
        })

        expect(body).to.exist.and.to.include.key("data")
        expect(body.data).to.include.keys("uuid", "tenant_uuid", "resource_version")
        expect(body.data.uuid).to.be.a("string").of.length.greaterThan(10)
        expect(body.data.tenant_uuid).to.eq(tid)
        expect(body.data.resource_version).to.be.a("string").of.length.greaterThan(5)

        expect(body.data.subjects).to.deep.eq(subjects)
    })

    it("can be read", async () => {
        const tenant = await createTenant()
        const tid = tenant.uuid
        const subjects = await createSubjects(tid)

        // create
        const payload = genPayload({ subjects })
        const { data: created } = await rootGroupClient.create({
            params: { tenant: tid },
            payload,
        })
        const gid = created.data.uuid
        const generated = {
            uuid: created.data.uuid,
            tenant_uuid: created.data.tenant_uuid,
            resource_version: created.data.resource_version,
            full_identifier: payload.identifier + "@group." + tenant.identifier,
            origin: "iam",
            extensions: null,
        }

        // read
        const { data: read } = await rootGroupClient.read({
            params: { tenant: tid, group: gid },
        })

        expect(read.data).to.deep.eq({ ...payload, ...generated }, "must have generated fields")
        expect(read.data).to.deep.eq(
            created.data,
            "reading and creation responses should contain the same data",
        )
        expect(read.data.resource_version).to.be.a("string").of.length.greaterThan(5)

        expect(read.data.subjects).to.deep.eq(subjects)
    })

    it("can be updated", async () => {
        const tid = await createTenantId()

        // create
        const created = await createGroup(tid)

        // update
        const subjects = await createSubjects(tid)
        const payload = genPayload({
            resource_version: created.resource_version,
            subjects,
        })
        const params = { tenant: tid, group: created.uuid }
        const { data: updated } = await rootGroupClient.update({
            params,
            payload,
        })

        // read
        const { data: read } = await rootGroupClient.read({ params })

        expect(read.data).to.deep.eq(updated.data)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const group = await createGroup(tid)
        const gid = group.uuid

        // delete
        const params = { tenant: tid, group: gid }
        await rootGroupClient.delete({ params })

        // read
        await rootGroupClient.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const group = await createGroup(tid)
        const gid = group.uuid

        // delete
        const params = { tenant: tid }
        const { data: body } = await rootGroupClient.list({ params })

        expect(body.data).to.be.an("object").and.include.keys("uuids")
        expect(body.data.uuids).to.be.an("array").of.length(1) // if not 1, maybe groups are not filtered by tenants
        expect(body.data.uuids[0]).to.eq(gid)
    })

    it("can be deleted by the tenant deletion", async () => {
        const tid = await createTenantId()
        const group = await createGroup(tid)

        await rootTenantAPI.delete({ params: { tenant: tid } })

        const params = { tenant: tid, group: group.uuid }
        const opts = expectStatus(404)
        await rootGroupClient.read({ params, opts })
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        const params = { group: "no-such" }

        before("create tenant", async () => {
            params.tenant = await createTenantId()
        })

        it("cannot read, gets 404", async () => {
            await rootGroupClient.read({ params, opts })
        })

        it("cannot update, gets 404", async () => {
            await rootGroupClient.update({
                params,
                opts,
                payload: genTenantPayload(),
            })
        })

        it("cannot delete, gets 404", async () => {
            await rootGroupClient.delete({ params, opts })
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

            beforeEach("create group payload", async () => {
                const subjects = await createSubjects(params.tenant)
                payload = genPayload({ subjects })
            })

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ params, payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ params, opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const { data } = await rootGroupClient.create({
                    params,
                    payload,
                })
                const gid = data.data.uuid

                await unauth.read({
                    params: { ...params, group: gid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const { data } = await rootGroupClient.create({
                    params,
                    payload,
                })
                const gid = data.data.uuid
                await unauth.update({
                    params: { ...params, group: gid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const { data } = await rootGroupClient.create({
                    params,
                    payload,
                })
                const gid = data.data.uuid
                await unauth.delete({
                    params: { ...params, group: gid },
                    opts,
                })
            })
        }
    })
})
