import React from "react"
import { Row, Col, Form, Upload, Icon, Spin, Popconfirm, Tooltip, notification } from "antd"
import * as cs from "./constants"

class TheFormDef extends React.PureComponent {
    state = {
        confirmDirty: false
    }

    componentDidMount() {
        this.props.pageTitle("Media")
        this.fetchRecords()
    }

    fetchRecords = () => {
        this.props.modelRequest(cs.ModelMedia, cs.Routes.GetMedia, cs.MethodGet)
    }

    handleDeleteRecord = (record) => {
        this.props.modelRequest(cs.ModelMedia, cs.Routes.DeleteMedia, cs.MethodDelete, { id: record.id })
            .then(() => {
                notification["success"]({ placement: cs.MsgPosition, message: "Image deleted", description: `"${record.filename}" deleted` })

                // Reload the table.
                this.fetchRecords()
            }).catch(e => {
                notification["error"]({ message: "Error", description: e.message })
            })
    }

    handleInsertMedia = (record) => {
        // The insertMedia callback may be passed down by the invoker (Campaign)
        if(!this.props.insertMedia) {
            return false
        }
        
        this.props.insertMedia(record.uri)
        return false
    }

    onFileChange = (f) => {
        if(f.file.error && f.file.response && f.file.response.hasOwnProperty("message")) {
            notification["error"]({ message: "Error uploading file", description: f.file.response.message })
        } else if(f.file.status === "done") {
            this.fetchRecords()
        }

        return false
    }

    render() {
        const { getFieldDecorator } = this.props.form
        const formItemLayout = {
            labelCol: { xs: { span: 16 }, sm: { span: 4 } },
            wrapperCol: { xs: { span: 16 }, sm: { span: 10 } }
        }

        return (
            <Spin spinning={false}>
                <Form>
                    <Form.Item
                        {...formItemLayout}
                        label="Upload images">
                        <div className="dropbox">
                            {getFieldDecorator("file", {
                                valuePropName: "file",
                                getValueFromEvent: this.normFile,
                                rules: [{ required: true }]
                            })(
                                <Upload.Dragger
                                    name="file"
                                    action="/api/media"
                                    multiple={ true }
                                    listType="picture"
                                    onChange={ this.onFileChange }
                                    accept=".gif, .jpg, .jpeg, .png">
                                    <p className="ant-upload-drag-icon">
                                        <Icon type="inbox" />
                                    </p>
                                    <p className="ant-upload-text">Click or drag file here</p>
                                </Upload.Dragger>
                            )}
                        </div>
                    </Form.Item>
                </Form>

                <section className="gallery">
                    {this.props.media && this.props.media.map((record, i) =>
                        <div key={ i } className="image">
                            <a onClick={ () => {
                                this.handleInsertMedia(record);
                                if( this.props.onCancel ) {
                                    this.props.onCancel();
                                }
                            } }><img alt={ record.filename } src={ record.thumb_uri } /></a>
                            <div className="actions">
                                <Tooltip title="View" placement="bottom"><a role="button" href={ record.uri } target="_blank"><Icon type="login" /></a></Tooltip>
                                <Popconfirm title="Are you sure?" onConfirm={() => this.handleDeleteRecord(record)}>
                                    <Tooltip title="Delete" placement="bottom"><a role="button"><Icon type="delete" /></a></Tooltip>
                                </Popconfirm>
                            </div>
                            <div className="name" title={ record.filename }>{ record.filename }</div>
                        </div>
                    )}
                </section>
            </Spin>
        )
    }
}
const TheForm = Form.create()(TheFormDef)

class Media extends React.PureComponent {
    render() {
        return (
            <section className="content media">
                <Row>
                    <Col span={22}><h1>Images</h1></Col>
                    <Col span={2}>
                    </Col>
                </Row>

                <TheForm { ...this.props }
                    media={ this.props.data[cs.ModelMedia] }>
                </TheForm>
            </section>
        )
    }
}

export default Media
