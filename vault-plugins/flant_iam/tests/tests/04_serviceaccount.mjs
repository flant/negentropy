import { expect } from "chai"
import { API } from "./lib/api.mjs"
import { expectStatus, getClient, rootToken } from "./lib/client.mjs"
import {
    EndpointBuilder,
    genMultipassPayload,
    genServiceAccountPayload,
    SubTenantEntrypointBuilder,
} from "./lib/subtenant.mjs"
import { genTenantPayload, TenantEndpointBuilder } from "./lib/tenant.mjs"

//    /tenant/{tid}/service_account/{said}

describe("Service Account", function () {
    const rootClient = getClient(rootToken)
    const rootTenantAPI = new API(rootClient, new TenantEndpointBuilder())

    const entrypointBuilder = new SubTenantEntrypointBuilder("service_account")
    const rootServiceAccountClient = new API(rootClient, entrypointBuilder)

    function genPayload(override) {
        return genServiceAccountPayload(override)
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

    async function createServiceAccount(tid) {
        const payload = genPayload()
        const { data: body } = await rootServiceAccountClient.create({
            params: { tenant: tid },
            payload,
        })
        return body.data
    }

    it("can be created", async () => {
        const tid = await createTenantId()

        const { data: body } = await rootServiceAccountClient.create({
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
        const { data: created } = await rootServiceAccountClient.create({
            params: { tenant: tid },
            payload,
        })
        const said = created.data.uuid
        const generated = {
            uuid: created.data.uuid,
            tenant_uuid: created.data.tenant_uuid,
            resource_version: created.data.resource_version,
            full_identifier:
                payload.identifier + "@service_account." + tenant.identifier,
            origin: "iam",
            extensions: null,
        }

        // read
        const { data: read } = await rootServiceAccountClient.read({
            params: { tenant: tid, service_account: said },
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
        const created = await createServiceAccount(tid)

        // update
        const payload = genPayload({
            resource_version: created.resource_version,
        })
        const params = { tenant: tid, service_account: created.uuid }
        const { data: updated } = await rootServiceAccountClient.update({
            params,
            payload,
        })

        // read
        const { data: read } = await rootServiceAccountClient.read({ params })
        const sub = read.data

        expect(read.data).to.deep.eq(updated.data)
    })

    it("can be deleted", async () => {
        const tid = await createTenantId()

        // create
        const serviceAccount = await createServiceAccount(tid)
        const said = serviceAccount.uuid

        // delete
        const params = { tenant: tid, service_account: said }
        await rootServiceAccountClient.delete({ params })

        // read
        await rootServiceAccountClient.read({ params, opts: expectStatus(404) })
    })

    it("can be listed", async () => {
        // create
        const tid = await createTenantId()
        const serviceAccount = await createServiceAccount(tid)
        const said = serviceAccount.uuid

        // delete
        const params = { tenant: tid }
        const { data: body } = await rootServiceAccountClient.list({ params })

        expect(body.data).to.be.an("object").and.include.keys("uuids")
        expect(body.data.uuids).to.be.an("array").of.length(1) // if not 1, maybe serviceAccounts are not filtered by tenants
        expect(body.data.uuids[0]).to.eq(said)
    })

    it("can be deleted by the tenant deletion", async () => {
        const tid = await createTenantId()
        const serviceAccount = await createServiceAccount(tid)

        await rootTenantAPI.delete({ params: { tenant: tid } })

        const params = { tenant: tid, service_account: serviceAccount.uuid }
        const opts = expectStatus(404)
        await rootServiceAccountClient.read({ params, opts })
    })

    describe("when does not exist", () => {
        const opts = expectStatus(404)
        const params = { service_account: "no-such" }

        before("create tenant", async () => {
            params.tenant = await createTenantId()
        })

        it("cannot read, gets 404", async () => {
            await rootServiceAccountClient.read({ params, opts })
        })

        it("cannot update, gets 404", async () => {
            await rootServiceAccountClient.update({
                params,
                opts,
                payload: genTenantPayload(),
            })
        })

        it("cannot delete, gets 404", async () => {
            await rootServiceAccountClient.delete({ params, opts })
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

            beforeEach("create serviceAccount payload", async () => {
                payload = genPayload()
            })

            it(`cannot create, gets ${expectedStatus}`, async () => {
                await unauth.create({ params, payload, opts })
            })

            it(`cannot list, gets ${expectedStatus}`, async () => {
                await unauth.list({ params, opts })
            })

            it(`cannot read, gets ${expectedStatus}`, async () => {
                const { data } = await rootServiceAccountClient.create({
                    params,
                    payload,
                })
                const said = data.data.uuid

                await unauth.read({
                    params: { ...params, service_account: said },
                    opts,
                })
            })

            it(`cannot update, gets ${expectedStatus}`, async () => {
                const { data } = await rootServiceAccountClient.create({
                    params,
                    payload,
                })
                const said = data.data.uuid
                await unauth.update({
                    params: { ...params, service_account: said },
                    payload,
                    opts,
                })
            })

            it(`cannot delete, gets ${expectedStatus}`, async () => {
                const { data } = await rootServiceAccountClient.create({
                    params,
                    payload,
                })
                const said = data.data.uuid
                await unauth.delete({
                    params: { ...params, service_account: said },
                    opts,
                })
            })
        }
    })

    describe("multipass", function () {
        const endpointBuilder = new EndpointBuilder([
            "tenant",
            "service_account",
            "multipass",
        ])
        const rootMPClient = new API(rootClient, endpointBuilder)

        async function createMultipass(t, u, override = {}) {
            const payload = genMultipassPayload(override)
            const params = { tenant: t, service_account: u }
            const { data } = await rootMPClient.create({ params, payload })
            return data.data
        }

        it("can be created", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)

            await createMultipass(t.uuid, sa.uuid)
        })

        it("contains expected data when created", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)

            const mp = await createMultipass(t.uuid, sa.uuid)

            expect(mp)
                .to.be.an("object")
                .and.include.keys(
                    "allowed_cidrs",
                    "allowed_roles",
                    "description",
                    "max_ttl",
                    "owner_type",
                    "owner_uuid",
                    "tenant_uuid",
                    "origin",
                    "extensions",
                    "ttl",
                    "uuid",
                    "valid_till",
                )
                .and.not.include.keys("salt")

            expect(mp.uuid, "uuid").to.be.a("string")
            expect(mp.owner_type, "owner_type").to.be.a("string")
            expect(mp.owner_uuid, "owner_uuid").to.be.a("string")
            expect(mp.tenant_uuid, "tenant_uuid").to.be.a("string")

            expect(mp.salt, "salt").to.be.undefined

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
            const sa = await createServiceAccount(t.uuid)
            const created = await createMultipass(t.uuid, sa.uuid)

            const params = {
                tenant: t.uuid,
                service_account: sa.uuid,
                multipass: created.uuid,
            }
            const { data } = await rootMPClient.read({ params })
            const read = data.data

            expect(read).to.deep.eq(created)
        })

        it("can be listed", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)

            const createId = () =>
                createMultipass(t.uuid, sa.uuid).then((mp) => mp.uuid)

            const ids = await Promise.all([createId(), createId(), createId()])

            const params = {
                tenant: t.uuid,
                service_account: sa.uuid,
            }

            const { data } = await rootMPClient.list({ params })

            expect(data.data.uuids).to.have.all.members(ids)
        })

        it("can be deleted", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)
            const created = await createMultipass(t.uuid, sa.uuid)

            const params = {
                tenant: t.uuid,
                service_account: sa.uuid,
                multipass: created.uuid,
            }

            await rootMPClient.delete({ params })

            await rootMPClient.read({ params, opts: expectStatus(404) })
        })

        it("cannot be updated", async () => {
            const t = await createTenant()
            const sa = await createServiceAccount(t.uuid)
            const createdMP = await createMultipass(t.uuid, sa.uuid)

            const params = {
                tenant: t.uuid,
                service_account: sa.uuid,
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
