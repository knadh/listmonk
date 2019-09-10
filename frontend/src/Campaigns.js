import React from "react"
import { Link } from "react-router-dom"
import {
  Row,
  Col,
  Button,
  Table,
  Icon,
  Tooltip,
  Tag,
  Popconfirm,
  Progress,
  Modal,
  notification,
  Input
} from "antd"
import dayjs from "dayjs"
import relativeTime from "dayjs/plugin/relativeTime"

import ModalPreview from "./ModalPreview"
import * as cs from "./constants"

class Campaigns extends React.PureComponent {
  defaultPerPage = 20

  state = {
    formType: null,
    pollID: -1,
    queryParams: {},
    stats: {},
    record: null,
    previewRecord: null,
    cloneName: "",
    cloneModalVisible: false,
    modalWaiting: false
  }

  // Pagination config.
  paginationOptions = {
    hideOnSinglePage: false,
    showSizeChanger: true,
    showQuickJumper: true,
    defaultPageSize: this.defaultPerPage,
    pageSizeOptions: ["20", "50", "70", "100"],
    position: "both",
    showTotal: (total, range) => `${range[0]} to ${range[1]} of ${total}`
  }

  constructor(props) {
    super(props)

    this.columns = [
      {
        title: "Name",
        dataIndex: "name",
        sorter: true,
        width: "20%",
        vAlign: "top",
        filterIcon: filtered => (
          <Icon
            type="search"
            style={{ color: filtered ? "#1890ff" : undefined }}
          />
        ),
        filterDropdown: ({
          setSelectedKeys,
          selectedKeys,
          confirm,
          clearFilters
        }) => (
          <div style={{ padding: 8 }}>
            <Input
              ref={node => {
                this.searchInput = node
              }}
              placeholder={`Search`}
              onChange={e =>
                setSelectedKeys(e.target.value ? [e.target.value] : [])
              }
              onPressEnter={() => confirm()}
              style={{ width: 188, marginBottom: 8, display: "block" }}
            />
            <Button
              type="primary"
              onClick={() => confirm()}
              icon="search"
              size="small"
              style={{ width: 90, marginRight: 8 }}
            >
              Search
            </Button>
            <Button
              onClick={() => {
                clearFilters()
              }}
              size="small"
              style={{ width: 90 }}
            >
              Reset
            </Button>
          </div>
        ),
        render: (text, record) => {
          const out = []
          out.push(
            <div className="name" key={`name-${record.id}`}>
              <Link to={`/campaigns/${record.id}`}>{text}</Link>
              <br />
              <span className="text-tiny">{record.subject}</span>
            </div>
          )

          if (record.tags.length > 0) {
            for (let i = 0; i < record.tags.length; i++) {
              out.push(<Tag key={`tag-${i}`}>{record.tags[i]}</Tag>)
            }
          }

          return out
        }
      },
      {
        title: "Status",
        dataIndex: "status",
        className: "status",
        width: "10%",
        filters: [
          { text: "Draft", value: "draft" },
          { text: "Running", value: "running" },
          { text: "Scheduled", value: "scheduled" },
          { text: "Paused", value: "paused" },
          { text: "Cancelled", value: "cancelled" },
          { text: "Finished", value: "finished" }
        ],
        render: (status, record) => {
          let color = cs.CampaignStatusColors.hasOwnProperty(status)
            ? cs.CampaignStatusColors[status]
            : ""
          return (
            <div>
              <Tag color={color}>{status}</Tag>
              {record.send_at && (
                <span className="text-tiny date">
                  Scheduled &mdash;{" "}
                  {dayjs(record.send_at).format(cs.DateFormat)}
                </span>
              )}
            </div>
          )
        }
      },
      {
        title: "Lists",
        dataIndex: "lists",
        width: "25%",
        align: "left",
        className: "lists",
        render: (lists, record) => {
          const out = []
          lists.forEach(l => {
            out.push(
              <Tag className="name" key={`name-${l.id}`}>
                <Link to={`/subscribers/lists/${l.id}`}>{l.name}</Link>
              </Tag>
            )
          })

          return out
        }
      },
      {
        title: "Stats",
        className: "stats",
        width: "30%",
        render: (text, record) => {
          if (
            record.status !== cs.CampaignStatusDraft &&
            record.status !== cs.CampaignStatusScheduled
          ) {
            return this.renderStats(record)
          }
        }
      },
      {
        title: "",
        dataIndex: "actions",
        className: "actions",
        width: "15%",
        render: (text, record) => {
          return (
            <div className="actions">
              {record.status === cs.CampaignStatusPaused && (
                <Popconfirm
                  title="Are you sure?"
                  onConfirm={() =>
                    this.handleUpdateStatus(record, cs.CampaignStatusRunning)
                  }
                >
                  <Tooltip title="Resume campaign" placement="bottom">
                    <a role="button">
                      <Icon type="rocket" />
                    </a>
                  </Tooltip>
                </Popconfirm>
              )}

              {record.status === cs.CampaignStatusRunning && (
                <Popconfirm
                  title="Are you sure?"
                  onConfirm={() =>
                    this.handleUpdateStatus(record, cs.CampaignStatusPaused)
                  }
                >
                  <Tooltip title="Pause campaign" placement="bottom">
                    <a role="button">
                      <Icon type="pause-circle-o" />
                    </a>
                  </Tooltip>
                </Popconfirm>
              )}

              {/* Draft with send_at */}
              {record.status === cs.CampaignStatusDraft && record.send_at && (
                <Popconfirm
                  title="The campaign will start automatically at the scheduled date and time. Schedule now?"
                  onConfirm={() =>
                    this.handleUpdateStatus(record, cs.CampaignStatusScheduled)
                  }
                >
                  <Tooltip title="Schedule campaign" placement="bottom">
                    <a role="button">
                      <Icon type="clock-circle" />
                    </a>
                  </Tooltip>
                </Popconfirm>
              )}

              {record.status === cs.CampaignStatusDraft && !record.send_at && (
                <Popconfirm
                  title="Campaign properties cannot be changed once it starts. Start now?"
                  onConfirm={() =>
                    this.handleUpdateStatus(record, cs.CampaignStatusRunning)
                  }
                >
                  <Tooltip title="Start campaign" placement="bottom">
                    <a role="button">
                      <Icon type="rocket" />
                    </a>
                  </Tooltip>
                </Popconfirm>
              )}

              {(record.status === cs.CampaignStatusPaused ||
                record.status === cs.CampaignStatusRunning) && (
                <Popconfirm
                  title="Are you sure?"
                  onConfirm={() =>
                    this.handleUpdateStatus(record, cs.CampaignStatusCancelled)
                  }
                >
                  <Tooltip title="Cancel campaign" placement="bottom">
                    <a role="button">
                      <Icon type="close-circle-o" />
                    </a>
                  </Tooltip>
                </Popconfirm>
              )}

              <Tooltip title="Preview campaign" placement="bottom">
                <a
                  role="button"
                  onClick={() => {
                    this.handlePreview(record)
                  }}
                >
                  <Icon type="search" />
                </a>
              </Tooltip>

              <Tooltip title="Clone campaign" placement="bottom">
                <a
                  role="button"
                  onClick={() => {
                    let r = {
                      ...record,
                      lists: record.lists.map(i => {
                        return i.id
                      })
                    }
                    this.handleToggleCloneForm(r)
                  }}
                >
                  <Icon type="copy" />
                </a>
              </Tooltip>

              {(record.status === cs.CampaignStatusDraft ||
                record.status === cs.CampaignStatusScheduled) && (
                <Popconfirm
                  title="Are you sure?"
                  onConfirm={() => this.handleDeleteRecord(record)}
                >
                  <Tooltip title="Delete campaign" placement="bottom">
                    <a role="button">
                      <Icon type="delete" />
                    </a>
                  </Tooltip>
                </Popconfirm>
              )}
            </div>
          )
        }
      }
    ]
  }

  progressPercent(record) {
    return Math.round(
      (this.getStatsField("sent", record) /
        this.getStatsField("to_send", record)) *
        100,
      2
    )
  }

  isDone(record) {
    return (
      this.getStatsField("status", record) === cs.CampaignStatusFinished ||
      this.getStatsField("status", record) === cs.CampaignStatusCancelled
    )
  }

  // getStatsField returns a stats field value of a given record if it
  // exists in the stats state, or the value from the record itself.
  getStatsField = (field, record) => {
    if (this.state.stats.hasOwnProperty(record.id)) {
      return this.state.stats[record.id][field]
    }

    return record[field]
  }

  renderStats = record => {
    let color = cs.CampaignStatusColors.hasOwnProperty(record.status)
      ? cs.CampaignStatusColors[record.status]
      : ""
    const startedAt = this.getStatsField("started_at", record)
    const updatedAt = this.getStatsField("updated_at", record)
    const sent = this.getStatsField("sent", record)
    const toSend = this.getStatsField("to_send", record)
    const isDone = this.isDone(record)

    const r = this.getStatsField("rate", record)
    const rate = r ? r : 0

    return (
      <div>
        {!isDone && (
          <Progress
            strokeColor={color}
            status="active"
            type="line"
            percent={this.progressPercent(record)}
          />
        )}
        <Row>
          <Col className="label" span={10}>
            Sent
          </Col>
          <Col span={12}>
            {sent >= toSend && <span>{toSend}</span>}
            {sent < toSend && (
              <span>
                {sent} / {toSend}
              </span>
            )}
            &nbsp;
            {record.status === cs.CampaignStatusRunning && (
              <Icon type="loading" style={{ fontSize: 12 }} spin />
            )}
          </Col>
        </Row>

        {rate > 0 && (
          <Row>
            <Col className="label" span={10}>
              Rate
            </Col>
            <Col span={12}>{Math.round(rate, 2)} / min</Col>
          </Row>
        )}

        <Row>
          <Col className="label" span={10}>
            Views
          </Col>
          <Col span={12}>{record.views}</Col>
        </Row>
        <Row>
          <Col className="label" span={10}>
            Clicks
          </Col>
          <Col span={12}>{record.clicks}</Col>
        </Row>
        <br />
        <Row>
          <Col className="label" span={10}>
            Created
          </Col>
          <Col span={12}>{dayjs(record.created_at).format(cs.DateFormat)}</Col>
        </Row>

        {startedAt && (
          <Row>
            <Col className="label" span={10}>
              Started
            </Col>
            <Col span={12}>{dayjs(startedAt).format(cs.DateFormat)}</Col>
          </Row>
        )}
        {isDone && (
          <Row>
            <Col className="label" span={10}>
              Ended
            </Col>
            <Col span={12}>{dayjs(updatedAt).format(cs.DateFormat)}</Col>
          </Row>
        )}
        {startedAt && updatedAt && (
          <Row>
            <Col className="label" span={10}>
              Duration
            </Col>
            <Col className="duration" span={12}>
              {dayjs(updatedAt).from(dayjs(startedAt), true)}
            </Col>
          </Row>
        )}
      </div>
    )
  }

  componentDidMount() {
    this.props.pageTitle("Campaigns")
    dayjs.extend(relativeTime)
    this.fetchRecords()

    // Did we land here to start a campaign?
    let loc = this.props.route.location
    let state = loc.state
    if (state && state.hasOwnProperty("campaign")) {
      this.handleUpdateStatus(state.campaign, state.campaignStatus)
      delete state.campaign
      delete state.campaignStatus
      this.props.route.history.replace({ ...loc, state })
    }
  }

  componentWillUnmount() {
    window.clearInterval(this.state.pollID)
  }

  fetchRecords = params => {
    if (!params) {
      params = {}
    }
    let qParams = {
      page: this.state.queryParams.page,
      per_page: this.state.queryParams.per_page
    }

    // Avoid sending blank string where the enum check will fail.
    if (!params.status) {
      delete params.status
    }

    if (params) {
      qParams = { ...qParams, ...params }
    }

    this.props
      .modelRequest(
        cs.ModelCampaigns,
        cs.Routes.GetCampaigns,
        cs.MethodGet,
        qParams
      )
      .then(r => {
        this.setState({
          queryParams: {
            ...this.state.queryParams,
            total: this.props.data[cs.ModelCampaigns].total,
            per_page: this.props.data[cs.ModelCampaigns].per_page,
            page: this.props.data[cs.ModelCampaigns].page,
            query: this.props.data[cs.ModelCampaigns].query,
            status: params.status
          }
        })

        this.startStatsPoll()
      })
  }

  startStatsPoll = () => {
    window.clearInterval(this.state.pollID)
    this.setState({ stats: {} })

    // If there's at least one running campaign, start polling.
    let hasRunning = false
    this.props.data[cs.ModelCampaigns].results.forEach(c => {
      if (c.status === cs.CampaignStatusRunning) {
        hasRunning = true
        return
      }
    })

    if (!hasRunning) {
      return
    }

    // Poll for campaign stats.
    let pollID = window.setInterval(() => {
      this.props
        .request(cs.Routes.GetRunningCampaignStats, cs.MethodGet)
        .then(r => {
          // No more running campaigns.
          if (r.data.data.length === 0) {
            window.clearInterval(this.state.pollID)
            this.fetchRecords()
            return
          }

          let stats = {}
          r.data.data.forEach(s => {
            stats[s.id] = s
          })

          this.setState({ stats: stats })
        })
        .catch(e => {
          console.log(e.message)
        })
    }, 3000)

    this.setState({ pollID: pollID })
  }

  handleUpdateStatus = (record, status) => {
    this.props
      .modelRequest(
        cs.ModelCampaigns,
        cs.Routes.UpdateCampaignStatus,
        cs.MethodPut,
        { id: record.id, status: status }
      )
      .then(() => {
        notification["success"]({
          placement: cs.MsgPosition,
          message: `Campaign ${status}`,
          description: `"${record.name}" ${status}`
        })

        // Reload the table.
        this.fetchRecords()
      })
      .catch(e => {
        notification["error"]({
          placement: cs.MsgPosition,
          message: "Error",
          description: e.message
        })
      })
  }

  handleDeleteRecord = record => {
    this.props
      .modelRequest(
        cs.ModelCampaigns,
        cs.Routes.DeleteCampaign,
        cs.MethodDelete,
        { id: record.id }
      )
      .then(() => {
        notification["success"]({
          placement: cs.MsgPosition,
          message: "Campaign deleted",
          description: `"${record.name}" deleted`
        })

        // Reload the table.
        this.fetchRecords()
      })
      .catch(e => {
        notification["error"]({
          placement: cs.MsgPosition,
          message: "Error",
          description: e.message
        })
      })
  }

  handleToggleCloneForm = record => {
    this.setState({
      cloneModalVisible: !this.state.cloneModalVisible,
      record: record,
      cloneName: record.name
    })
  }

  handleCloneCampaign = record => {
    this.setState({ modalWaiting: true })
    this.props
      .modelRequest(
        cs.ModelCampaigns,
        cs.Routes.CreateCampaign,
        cs.MethodPost,
        record
      )
      .then(resp => {
        notification["success"]({
          placement: cs.MsgPosition,
          message: "Campaign created",
          description: `${record.name} created`
        })

        this.setState({ record: null, modalWaiting: false })
        this.props.route.history.push(
          cs.Routes.ViewCampaign.replace(":id", resp.data.data.id)
        )
      })
      .catch(e => {
        notification["error"]({
          placement: cs.MsgPosition,
          message: "Error",
          description: e.message
        })
        this.setState({ modalWaiting: false })
      })
  }

  handlePreview = record => {
    this.setState({ previewRecord: record })
  }

  render() {
    const pagination = {
      ...this.paginationOptions,
      ...this.state.queryParams
    }

    return (
      <section className="content campaigns">
        <Row>
          <Col xs={24} sm={14}>
            <h1>Campaigns</h1>
          </Col>
          <Col xs={24} sm={10} className="right header-action-break">
            <Link to="/campaigns/new">
              <Button type="primary" icon="plus" role="link">
                New campaign
              </Button>
            </Link>
          </Col>
        </Row>
        <br />

        <Table
          className="campaigns"
          columns={this.columns}
          rowKey={record => record.uuid}
          dataSource={(() => {
            if (
              !this.props.data[cs.ModelCampaigns] ||
              !this.props.data[cs.ModelCampaigns].hasOwnProperty("results")
            ) {
              return []
            }
            return this.props.data[cs.ModelCampaigns].results
          })()}
          loading={this.props.reqStates[cs.ModelCampaigns] !== cs.StateDone}
          pagination={pagination}
          onChange={(pagination, filters, sorter, records) => {
            this.fetchRecords({
              per_page: pagination.pageSize,
              page: pagination.current,
              status:
                filters.status && filters.status.length > 0
                  ? filters.status
                  : "",
              query:
                filters.name && filters.name.length > 0 ? filters.name[0] : ""
            })
          }}
        />

        {this.state.previewRecord && (
          <ModalPreview
            title={this.state.previewRecord.name}
            previewURL={cs.Routes.PreviewCampaign.replace(
              ":id",
              this.state.previewRecord.id
            )}
            onCancel={() => {
              this.setState({ previewRecord: null })
            }}
          />
        )}

        {this.state.cloneModalVisible && this.state.record && (
          <Modal
            visible={this.state.record !== null}
            width="500px"
            className="clone-campaign-modal"
            title={"Clone " + this.state.record.name}
            okText="Clone"
            confirmLoading={this.state.modalWaiting}
            onCancel={this.handleToggleCloneForm}
            onOk={() => {
              this.handleCloneCampaign({
                ...this.state.record,
                name: this.state.cloneName
              })
            }}
          >
            <Input
              autoFocus
              defaultValue={this.state.record.name}
              style={{ width: "100%" }}
              onChange={e => {
                this.setState({ cloneName: e.target.value })
              }}
            />
          </Modal>
        )}
      </section>
    )
  }
}

export default Campaigns
