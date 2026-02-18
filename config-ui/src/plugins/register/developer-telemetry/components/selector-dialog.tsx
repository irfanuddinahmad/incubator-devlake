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

import { useState, useEffect } from 'react';
import { Modal } from 'antd';
import MillerColumnsSelect from 'miller-columns-select';

import { Block, Loading } from '@/components';
import { request } from '@/utils';
import { TelemetryConnectionType } from '../types';

import * as S from '../styled';

interface Props {
  open: boolean;
  saving: boolean;
  onCancel: () => void;
  onSubmit: (items: TelemetryConnectionType[]) => void;
}

export const SelectorDialog = ({ open, saving, onCancel, onSubmit }: Props) => {
  const [selectedIds, setSelectedIds] = useState<ID[]>([]);
  const [connections, setConnections] = useState<TelemetryConnectionType[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      const loadConnections = async () => {
        setLoading(true);
        try {
          const res = await request('/plugins/developer_telemetry/connections');
          setConnections(res);
        } catch (error) {
          console.error('Failed to load connections:', error);
        } finally {
          setLoading(false);
        }
      };
      loadConnections();
    }
  }, [open]);

  const handleSubmit = () => onSubmit(connections.filter((it) => selectedIds.includes(it.id)));

  return (
    <Modal
      open={open}
      width={820}
      centered
      title="Manage Connections: Developer Telemetry"
      okText="Confirm"
      okButtonProps={{
        disabled: !selectedIds.length,
        loading: saving,
      }}
      onCancel={onCancel}
      onOk={handleSubmit}
    >
      <S.Wrapper>
        <Block
          title="Telemetry Connections"
          description="Select an existing Developer Telemetry connection to import to the current project."
        >
          <MillerColumnsSelect
            columnCount={1}
            columnHeight={160}
            getHasMore={() => false}
            renderLoading={() => <Loading size={20} style={{ padding: '4px 12px' }} />}
            items={connections.map((it) => ({
              parentId: null,
              id: it.id,
              title: it.name,
              name: it.name,
            }))}
            selectedIds={selectedIds}
            onSelectItemIds={setSelectedIds}
          />
        </Block>
      </S.Wrapper>
    </Modal>
  );
};

export default SelectorDialog;
