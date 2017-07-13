#!/usr/bin/python
# Copyright 2017 Northern.tech AS
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        https://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
import io

from uuid import uuid4

import bravado
import requests

from client import SimpleArtifactsClient, ArtifactsClientError, \
    DeploymentsClient, InventoryClient, SimpleDeviceClient
from common import artifact_from_data, Device


class TestDeployment(DeploymentsClient):

    def setup(self):
        self.setup_swagger()

    @staticmethod
    def inventory_add_dev(dev):
        inv = InventoryClient()
        inv.report_attributes(dev.fake_token, [
            {
                'name': 'device_type',
                'value': dev.device_type,
            },
        ])

    def test_deployments_get(self):
        res = self.client.deployments.get_deployments(Authorization='foo').result()
        self.log.debug('result: %s', res)

        # try with bogus image ID
        try:
            res = self.client.deployments.get_deployments_id(Authorization='foo',
                                                         id='foo').result()
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == 400
        else:
            raise AssertionError('expected to fail')

    def test_deployments_new_bogus(self):

        # NOTE: cannot make requests with arbitary data through swagger client,
        # so we'll use requests directly instead
        rsp = requests.post(self.make_api_url('/deployments'), data='foobar')
        assert 400 <= rsp.status_code < 500
        # some broken JSON now
        rsp = requests.post(self.make_api_url('/deployments'), data='{"foo": }',
                            headers={'Content-Type': 'application/json'})
        assert 400 <= rsp.status_code < 500

        baddeps = [
            self.make_new_deployment(name='foobar', artifact_name='someartifact', devices=[]),
            self.make_new_deployment(name='', artifact_name='someartifact', devices=['foo']),
            self.make_new_deployment(name='adad', artifact_name='', devices=['foo']),
            self.make_new_deployment(name='', artifact_name='', devices=['foo']),
        ]
        for newdep in baddeps:
            # try bogus image data
            try:
                res = self.client.deployments.post_deployments(Authorization='foo',
                                                               deployment=newdep).result()
            except bravado.exception.HTTPError as e:
                assert e.response.status_code == 400
            else:
                raise AssertionError('expected to fail')

    def test_deployments_new_valid(self):
        """Add a new valid deployment, verify its status, verify device deployment
        status, abort and verify eveything once again"""
        dev = Device()

        self.log.info('fake device with ID: %s', dev.devid)

        self.inventory_add_dev(dev)

        data = b'foo_bar'
        artifact_name = 'hammer-update ' + str(uuid4())
        # come up with an artifact
        with artifact_from_data(name=artifact_name, data=data, devicetype=dev.device_type) as art:
            ac = SimpleArtifactsClient()
            artid = ac.add_artifact(description='some description', size=art.size,
                                    data=art)

            newdep = self.make_new_deployment(name='fake deployment', artifact_name=artifact_name,
                                              devices=[dev.devid])
            depid = self.add_deployment(newdep)

            # artifact is used in deployment, so attempts to remove it should
            # fail
            try:
                ac.delete_artifact(artid)
            except ArtifactsClientError as ace:
                #  artifact is used in deployment
                assert ace.response.status_code == 409
            else:
                raise AssertionError('expected to fail')

            dep = self.client.deployments.get_deployments_id(Authorization='foo',
                                                             id=depid).result()[0]
            self.log.debug('deployment dep: %s', dep)
            assert dep.artifact_name == artifact_name
            assert dep.id == depid
            assert dep.status == 'pending'

            # fetch device status
            depdevs = self.client.deployments.get_deployments_deployment_id_devices(Authorization='foo',
                                                                         deployment_id=depid).result()[0]
            self.log.debug('deployment devices: %s', depdevs)
            assert len(depdevs) == 1
            depdev = depdevs[0]
            assert depdev.status == 'pending'
            assert depdev.id == dev.devid

            # verify statistics
            self.verify_deployment_stats(depid, expected={
                'pending': 1,
            })

            # abort deployment
            self.abort_deployment(depid)

            # that it's 'finished' now
            aborted_dep = self.client.deployments.get_deployments_id(Authorization='foo',
                                                             id=depid).result()[0]
            self.log.debug('deployment dep: %s', aborted_dep)
            assert aborted_dep.status == 'finished'

            # verify statistics once again
            self.verify_deployment_stats(depid, expected={
                'aborted': 1,
            })

            # fetch device status
            depdevs = self.client.deployments.get_deployments_deployment_id_devices(Authorization='foo',
                                                                         deployment_id=depid).result()[0]
            self.log.debug('deployment devices: %s', depdevs)
            assert len(depdevs) == 1
            depdev = depdevs[0]
            assert depdev.status == 'aborted'

            # deleting artifact should succeed
            ac.delete_artifact(artid)

    def test_deployments_new_no_artifact(self):
        """Try to add deployment without an artifact, verify that it failed with 422"""
        dev = Device()

        self.log.info('fake device with ID: %s', dev.devid)

        self.inventory_add_dev(dev)

        artifact_name = 'no-artifact ' + str(uuid4())
        # come up with an artifact
        newdep = self.make_new_deployment(name='fake deployment', artifact_name=artifact_name,
                                          devices=[dev.devid])
        try:
            self.add_deployment(newdep)
        except bravado.exception.HTTPError as err:
            assert err.response.status_code == 422
        else:
            raise AssertionError('expected to fail')


    def test_device_deployments_simple(self):
        """Check that device can get next deployment, simple cases:
        - bogus token
        - valid update
        - device type incompatible with artifact
        """
        dev = Device()

        self.log.info('fake device with ID: %s', dev.devid)

        self.inventory_add_dev(dev)

        data = b'foo_bar'
        artifact_name = 'hammer-update ' + str(uuid4())
        # come up with an artifact
        with artifact_from_data(name=artifact_name, data=data, devicetype=dev.device_type) as art:
            ac = SimpleArtifactsClient()
            with ac.with_added_artifact(description='desc', size=art.size, data=art) as artid:

                newdep = self.make_new_deployment(name='foo', artifact_name=artifact_name,
                                                  devices=[dev.devid])

                with self.with_added_deployment(newdep) as depid:
                    dc = SimpleDeviceClient()
                    self.log.debug('device token %s', dev.fake_token)

                    # try with some bogus token
                    try:
                        dc.get_next_deployment('foo-bar-baz', artifact_name=artifact_name,
                                                     device_type=dev.device_type)
                    except bravado.exception.HTTPError as err:
                        assert 400 <= err.response.status_code < 500
                    else:
                        raise AssertionError('expected to fail')

                    # pretend we have another artifact installed
                    nextdep = dc.get_next_deployment(dev.fake_token,
                                                     artifact_name='different {}'.format(artifact_name),
                                                     device_type=dev.device_type)
                    self.log.info('device next: %s', nextdep)
                    assert nextdep

                    assert dev.device_type in nextdep.artifact['device_types_compatible']

                    # pretend our device type is different than expected
                    nextdep = dc.get_next_deployment(dev.fake_token,
                                                     artifact_name='different {}'.format(artifact_name),
                                                     device_type='other {}'.format(dev.device_type))
                    self.log.info('device next: %s', nextdep)
                    assert nextdep == None
                    # verify that device status was properly recorded
                    self.verify_deployment_stats(depid, expected={
                        'noartifact': 1,
                    })

    def test_device_deployments_already_installed(self):
        """Check case with already installed artifact
        """
        dev = Device()

        self.log.info('fake device with ID: %s', dev.devid)

        self.inventory_add_dev(dev)

        data = b'foo_bar'
        artifact_name = 'hammer-update ' + str(uuid4())
        # come up with an artifact
        with artifact_from_data(name=artifact_name, data=data, devicetype=dev.device_type) as art:
            ac = SimpleArtifactsClient()
            with ac.with_added_artifact(description='desc', size=art.size, data=art) as artid:

                newdep = self.make_new_deployment(name='foo', artifact_name=artifact_name,
                                                  devices=[dev.devid])

                with self.with_added_deployment(newdep) as depid:
                    dc = SimpleDeviceClient()
                    self.log.debug('device token %s', dev.fake_token)

                    # pretend we have the same artifact installed already
                    # NOTE: asking for a deployment while having it already
                    # installed is special in the sense that the status of
                    # deployment for this device will be marked as 'already-installed'
                    nextdep = dc.get_next_deployment(dev.fake_token, artifact_name=artifact_name,
                                                     device_type=dev.device_type)
                    self.log.info('device next: %s', nextdep)
                    assert nextdep == None
                    # verify that device status was properly recorded
                    self.verify_deployment_stats(depid, expected={
                        'already-installed': 1,
                    })

    def test_device_deployments_full(self):
        """Check that device can get next deployment, full cycle
        """
        dev = Device()

        self.log.info('fake device with ID: %s', dev.devid)

        self.inventory_add_dev(dev)

        data = b'foo_bar'
        artifact_name = 'hammer-update ' + str(uuid4())
        # come up with an artifact
        with artifact_from_data(name=artifact_name, data=data, devicetype=dev.device_type) as art:
            ac = SimpleArtifactsClient()
            with ac.with_added_artifact(description='desc', size=art.size, data=art) as artid:

                newdep = self.make_new_deployment(name='foo', artifact_name=artifact_name,
                                                  devices=[dev.devid])

                with self.with_added_deployment(newdep) as depid:
                    dc = SimpleDeviceClient()
                    self.log.debug('device token %s', dev.fake_token)

                    self.verify_deployment_stats(depid, expected={
                        'pending': 1,
                    })

                    # pretend we have another artifact installed
                    nextdep = dc.get_next_deployment(dev.fake_token,
                                                     artifact_name='different {}'.format(artifact_name),
                                                     device_type=dev.device_type)
                    self.log.info('device next: %s', nextdep)
                    assert nextdep

                    assert dev.device_type in nextdep.artifact['device_types_compatible']

                    for st in ['downloading', 'installing', 'rebooting']:
                        dc.report_status(token=dev.fake_token, devdepid=nextdep.id, status=st)
                        self.verify_deployment_stats(depid, expected={
                            st: 1,
                        })

                    # we have reported some statuses now, but not the final
                    # status, try to get the next deployment
                    againdep = dc.get_next_deployment(dev.fake_token,
                                                     artifact_name='different {}'.format(artifact_name),
                                                     device_type=dev.device_type)
                    assert againdep
                    assert againdep.id == nextdep.id

                    # deployment should be marked as inprogress
                    dep = self.client.deployments.get_deployments_id(Authorization='foo',
                                                                     id=depid).result()[0]
                    assert dep.status == 'inprogress'

                    # report final status
                    dc.report_status(token=dev.fake_token, devdepid=nextdep.id, status='success')
                    self.verify_deployment_stats(depid, expected={
                        'success': 1,
                    })

                    dep = self.client.deployments.get_deployments_id(Authorization='foo',
                                                                     id=depid).result()[0]
                    assert dep.status == 'finished'

                    # report failure as final status
                    dc.report_status(token=dev.fake_token, devdepid=nextdep.id, status='failure')
                    self.verify_deployment_stats(depid, expected={
                        'failure': 1,
                    })

                    # deployment is finished, should get no more updates
                    nodep = dc.get_next_deployment(dev.fake_token,
                                                     artifact_name='other {}'.format(artifact_name),
                                                     device_type=dev.device_type)
                    assert nodep == None

                    # as a joke, report rebooting now
                    dc.report_status(token=dev.fake_token, devdepid=nextdep.id, status='rebooting')
                    self.verify_deployment_stats(depid, expected={
                        'rebooting': 1,
                    })
                    # deployment is in progress again
                    dep = self.client.deployments.get_deployments_id(Authorization='foo',
                                                                     id=depid).result()[0]
                    assert dep.status == 'inprogress'

                    # go on, let's pretend that the artifact is already installed
                    nodep = dc.get_next_deployment(dev.fake_token,
                                                     artifact_name=artifact_name,
                                                     device_type=dev.device_type)
                    assert nodep == None
                    self.verify_deployment_stats(depid, expected={
                        'already-installed': 1,
                    })

    def test_device_deployments_logs(self):
        """Check that device can get next deployment, full cycle
        """
        dev = Device()

        self.log.info('fake device with ID: %s', dev.devid)

        self.inventory_add_dev(dev)

        data = b'foo_bar'
        artifact_name = 'hammer-update ' + str(uuid4())
        # come up with an artifact
        with artifact_from_data(name=artifact_name, data=data, devicetype=dev.device_type) as art:
            ac = SimpleArtifactsClient()
            with ac.with_added_artifact(description='desc', size=art.size, data=art) as artid:

                newdep = self.make_new_deployment(name='foo', artifact_name=artifact_name,
                                                  devices=[dev.devid])

                with self.with_added_deployment(newdep) as depid:
                    dc = SimpleDeviceClient()
                    self.log.debug('device token %s', dev.fake_token)

                    # pretend we have another artifact installed
                    nextdep = dc.get_next_deployment(dev.fake_token,
                                                     artifact_name='different {}'.format(artifact_name),
                                                     device_type=dev.device_type)
                    self.log.info('device next: %s', nextdep)
                    assert nextdep

                    dc.upload_logs(dev.fake_token, nextdep.id, logs=[
                        'foo bar baz',
                        'lorem ipsum',
                    ])

                    rsp = self.client.deployments.get_deployments_deployment_id_devices_device_id_log(
                        Authorization='foo',
                        deployment_id=depid,
                        device_id=dev.devid).result()[1]
                    logs = rsp.text
                    self.log.info('device logs\n%s', logs)

                    assert 'lorem ipsum' in logs
                    assert 'foo bar baz' in logs
