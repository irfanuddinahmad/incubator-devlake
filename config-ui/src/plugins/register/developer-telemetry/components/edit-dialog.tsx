/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { useState, useMemo, useEffect } from 'react';
import { Modal, Input } from 'antd';

import { Block } from '@/components';
import { request, operator } from '@/utils';
import { TelemetryConnectionType } from '../types';

interface Props {
  initialId: ID;
  connections: TelemetryConnectionType[];
  onCancel: () => void;
}

export const EditDialog = ({ initialId, connections, onCancel }: Props) => {
  const connection = useMemo(() => connections.find((c) => c.id === initialId), [initialId, connections]);
  const [operating, setOperating] = useState(false);
  const [name, setName] = useState('');

  useEffect(() => {
    if (connection) {
      setName(connection.name);
    }
  }, [connection]);

  const handleSubmit = async () => {
    const [success] = await operator(
      async () => {
        await request(`/plugins/developer_telemetry/connections/${initialId}`, {
          method: 'PATCH',
          data: {
            name,
          },
        });
      },
      {
        setOperating,
      },
    );

    if (success) {
      onCancel();
    }
  };

  if (!connection) {
    return null;
  }

  return (
    <Modal
      open
      width={820}
      centered
      title="Edit Connection Name"
      okText="Save"
      okButtonProps={{
        disabled: !name,
        loading: operating,
      }}
      onCancel={onCancel}
      onOk={handleSubmit}
    >
      <Block title="Name" required>
        <Input value={name} onChange={(e) => setName(e.target.value)} />
      </Block>
    </Modal>
  );
};

export default EditDialog;
