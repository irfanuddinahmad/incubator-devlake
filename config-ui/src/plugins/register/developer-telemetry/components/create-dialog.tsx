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

import { useState } from 'react';
import { CheckCircleOutlined } from '@ant-design/icons';
import { Modal, Input, InputNumber } from 'antd';

import { Block, CopyText, ExternalLink } from '@/components';
import { request, operator } from '@/utils';
import { formatReportEndpoint } from './utils';

import * as S from '../styled';

interface Props {
  open: boolean;
  onCancel: () => void;
  onSubmitAfter?: (id: ID) => void;
}

export const CreateDialog = ({ open, onCancel, onSubmitAfter }: Props) => {
  const [operating, setOperating] = useState(false);
  const [step, setStep] = useState(1);
  const [name, setName] = useState('');
  const [record, setRecord] = useState({
    id: 0,
    reportEndpoint: '',
    apiKey: '',
    endpoint: '',
  });

  const handleSubmit = async () => {
    if (step === 1) {
      const [success, res] = await operator(
        async () => {
          const response = await request('/plugins/developer_telemetry/connections', {
            method: 'POST',
            data: {
              name,
              endpoint: window.location.origin,
              proxy: '',
              rateLimitPerHour: 10000,
            },
          });
          return response;
        },
        {
          setOperating,
          hideToast: true,
        },
      );

      if (success && res) {
        setStep(2);
        setRecord({
          id: res.id,
          apiKey: res.apiKey?.apiKey || '',
          endpoint: res.endpoint || window.location.origin,
          reportEndpoint: formatReportEndpoint(
            res.endpoint || window.location.origin,
            res.id,
            res.apiKey?.apiKey || '',
          ),
        });
        onSubmitAfter?.(res.id);
      }
    } else {
      onCancel();
    }
  };

  return (
    <Modal
      open={open}
      width={820}
      centered
      title="Add a New Developer Telemetry"
      footer={step === 2 ? null : undefined}
      okText={step === 1 ? 'Generate POST URL' : 'Done'}
      okButtonProps={{
        disabled: step === 1 && !name,
        loading: operating,
      }}
      onCancel={onCancel}
      onOk={handleSubmit}
    >
      {step === 1 && (
        <S.Wrapper>
          <Block
            title="Developer Telemetry Name"
            description="Give your Developer Telemetry a unique name to help you identify it in the future."
            required
          >
            <Input placeholder="Developer Telemetry Name" value={name} onChange={(e) => setName(e.target.value)} />
          </Block>
        </S.Wrapper>
      )}
      {step === 2 && (
        <S.Wrapper>
          <h2>
            <CheckCircleOutlined />
            <span>CURL commands generated. Please copy them now.</span>
          </h2>
          <p>
            A non-expired API key is automatically generated for the authentication of the developer telemetry. This key
            will only show now. You can revoke it in the developer telemetry page at any time.
          </p>
          <Block title="Telemetry Data">
            <h5>Post to send developer metrics</h5>
            <CopyText content={record.reportEndpoint} />
          </Block>
        </S.Wrapper>
      )}
    </Modal>
  );
};

export default CreateDialog;
