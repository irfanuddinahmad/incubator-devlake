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

import { useMemo } from 'react';
import { Modal } from 'antd';

import { Block, CopyText, ExternalLink } from '@/components';
import { TelemetryConnectionType } from '../types';
import { formatReportEndpoint } from './utils';

import * as S from '../styled';

interface Props {
  initialId: ID;
  connections: TelemetryConnectionType[];
  onCancel: () => void;
}

export const ViewDialog = ({ initialId, connections, onCancel }: Props) => {
  const connection = useMemo(() => connections.find((c) => c.id === initialId), [initialId, connections]);
  const prefix = useMemo(() => `${window.location.origin}`, []);

  if (!connection) {
    return null;
  }

  const reportEndpoint = formatReportEndpoint(prefix, connection.id, '{API_KEY}');

  return (
    <Modal open width={820} centered title="View Telemetry Connection" footer={null} onCancel={onCancel}>
      <S.Wrapper>
        <p>
          Copy the following CURL command to your telemetry collector to push developer metrics by making a POST to
          DevLake. Please replace the {'{'}API_KEY{'}'} with your actual API key.
        </p>
        <Block title="Report Endpoint">
          <h5>Post to send telemetry data</h5>
          <CopyText content={reportEndpoint} />
          <p>
            See the <ExternalLink link="https://github.com/apache/incubator-devlake">documentation</ExternalLink> for
            the full payload schema.
          </p>
        </Block>
        <p style={{ marginTop: 16, color: '#999' }}>
          <strong>Note:</strong> For security reasons, the API key is only shown once during connection creation and
          cannot be retrieved later. If you've lost your API key, please delete and recreate this connection.
        </p>
      </S.Wrapper>
    </Modal>
  );
};

export default ViewDialog;
