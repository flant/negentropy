import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import { genTenantPayload, TenantEndpointBuilder } from "./lib/tenant.mjs"
import { expect } from "chai"
import {
    genRoleBindingPayload,
    SubTenantEntrypointBuilder,
} from "./lib/subtenant.mjs"
import { API } from "./lib/api.mjs"

//    /tenant/{tid}/role_binding/{rbid}

describe("Role Binding", function () {
    const rootClient = getClient(rootToken)
    const rootTenantAPI = new API(rootClient, new TenantEndpointBuilder())

    const entrypointBuilder = new SubTenantEntrypointBuilder("role_binding")
    const rootRoleBindingClient = new API(rootClient, entrypointBuilder)

    function genPayload(override) {
        return genRoleBindingPayload(override)
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

    async function createRoleBinding(tid) {
        const payload = genPayload()
        const { data: body } = await rootRoleBindingClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data
    }

    it("can be created", async () => {
        const tid = await createTenantId()

        const { data: body } = await rootRoleBindingClient.create({
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
        const { data: created } = await rootRoleBindingClient.create({
            params: { tenant: tid },
            payload,
        })
        const rbid = created.data.uuid
        const generated = {
            uuid: created.data.uuid,
            tenant_uuid: created.data.tenant_uuid,
            resource_version: created.data.resource_version,
        }

        // read
        const params = { tenant: tid, role_binding: rbid }
        const { data: read } = await rootRoleBindingClient.read({ params })

        const subResp = { ...payload, ...generated }
        delete subResp.ttl
        console.log(read.data)
        expect(read.data).to.deep.contain(
            subResp,
            "must contain generated fields",
        )
        expect(read.data.resource_version)
            .to.be.a("string")
            .of.length.greaterThan(5)

        expect(read.data.valid_till).to.lt(Date.now() + payload.ttl)

        expect(read.data).to.deep.eq(
            created.data,
            "reading and creation responses should contain the same data",
        )
    })

    it("can be updated", async () => {
        const tid = await createTenantId()

        // create
        const created = await createRoleBinding(tid)

        // update
        const payload = genPayload({
            resource_version: created.resource_version,
        })
        const params = { tenant: tid, role_binding: created.uuid }
        const { data: updated } = await rootRoleBindingClient.update({
            params,
            payload,
        })

        // read
        const { data: read } = await rootRoleBindingClient.read({ params })
        const sub = read.data

        expect(read.data).to.deep.eq(updated.data)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const roleBinding = await createRoleBinding(tid)
        const rbid = roleBinding.uuid

        // delete
        const params = { tenant: tid, role_binding: rbid }
        await rootRoleBindingClient.delete({ params })

        // read
        await rootRoleBindingClient.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const roleBinding = await createRoleBinding(tid)
        const rbid = roleBinding.uuid

        // delete
        const params = { tenant: tid }
        const { data: body } = await rootRoleBindingClient.list({ params })

        expect(body.data).to.be.an("object").and.include.keys("uuids")
        expect(body.data.uuids).to.be.an("array").of.length(1) // if not 1, maybe roleBindings are not filtered by tenants
        expect(body.data.uuids[0]).to.eq(rbid)
    })

    it("can be deleted by the tenant deletion", async () => {
        const tid = await createTenantId()
        const roleBinding = await createRoleBinding(tid)

        await rootTenantAPI.delete({ params: { tenant: tid } })

        const params = { tenant: tid, role_binding: roleBinding.uuid }
        const opts = expectStatus(404)
        await rootRoleBindingClient.read({ params, opts })
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        const params = { role_binding: "no-such" }

        before("create tenant", async () => {
            params.tenant = await createTenantId()
        })

        it("cannot read, gets 404", async () => {
            await rootRoleBindingClient.read({ params, opts })
        })

        it("cannot update, gets 404", async () => {
            await rootRoleBindingClient.update({
                params,
                opts,
                payload: genTenantPayload(),
            })
        })

        it("cannot delete, gets 404", async () => {
            await rootRoleBindingClient.delete({ params, opts })
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

            beforeEach("create roleBinding payload", async () => {
                payload = genPayload()
            })

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ params, payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ params, opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const { data } = await rootRoleBindingClient.create({
                    params,
                    payload,
                })
                const rbid = data.data.uuid

                await unauth.read({
                    params: { ...params, role_binding: rbid },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const { data } = await rootRoleBindingClient.create({
                    params,
                    payload,
                })
                const rbid = data.data.uuid
                await unauth.update({
                    params: { ...params, role_binding: rbid },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const { data } = await rootRoleBindingClient.create({
                    params,
                    payload,
                })
                const rbid = data.data.uuid
                await unauth.delete({
                    params: { ...params, role_binding: rbid },
                    opts,
                })
            })
        }
    })
})
