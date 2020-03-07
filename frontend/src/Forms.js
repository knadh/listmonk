import React from "react"
import {
  Row,
  Col,
  Checkbox,
} from "antd"

import * as cs from "./constants"

class Forms extends React.PureComponent {
  state = {
    lists: [],
    selected: [],
    selectedUUIDs: [],
    indeterminate: false,
    checkAll: false
  }

  componentDidMount() {
    this.props.pageTitle("Subscription forms")
    this.props
      .modelRequest(cs.ModelLists, cs.Routes.GetLists, cs.MethodGet, {
        per_page: "all"
      })
      .then(() => {
        this.setState({ lists: this.props.data[cs.ModelLists].results })
      })
  }

  handleSelectAll = e => {
    const uuids = this.state.lists.map(l => l.uuid)
    this.setState({
      selectedUUIDs: e.target.checked ? uuids : [],
      indeterminate: false,
      checkAll: e.target.checked
    })
    this.handleSelection(e.target.checked ? uuids : [])
  }

  handleSelection(sel) {
    let out = []
    sel.forEach(s => {
      const item = this.state.lists.find(l => {
        return l.uuid === s
      })
      if (item) {
        out.push(item)
      }
    })

    this.setState({
      selected: out,
      selectedUUIDs: sel,
      indeterminate: sel.length > 0 && sel.length < this.state.lists.length,
      checkAll: sel.length === this.state.lists.length
    })
  }

  render() {
    return (
      <section className="content list-form">
        <h1>Subscription forms</h1>
        <hr />
        <Row gutter={[16, 40]}>
          <Col span={24} md={8}>
            <h2>Lists</h2>
            <Checkbox
              indeterminate={this.state.indeterminate}
              onChange={this.handleSelectAll}
              checked={this.state.checkAll}
            >
              Select all
            </Checkbox>
            <hr />
            <Checkbox.Group
              className="lists"
              options={this.state.lists.map(l => {
                return { label: l.name, value: l.uuid }
              })}
              onChange={sel => this.handleSelection(sel)}
              value={this.state.selectedUUIDs}
            />
          </Col>
          <Col span={24} md={16}>
            <h2>Form HTML</h2>
            <p>
              Use the following HTML to show a subscription form on an external
              webpage.
            </p>
            <p>
              The form should have the{" "}
              <code>
                <strong>email</strong>
              </code>{" "}
              field and one or more{" "}
              <code>
                <strong>l</strong>
              </code>{" "}
              (list UUID) fields. The{" "}
              <code>
                <strong>name</strong>
              </code>{" "}
              field is optional.
            </p>
            <pre className="html">
              {`<form method="post" action="${
                window.CONFIG.rootURL
              }/subscription/form" class="listmonk-form">
    <div>
        <h3>Subscribe</h3>
        <p><input type="text" name="email" placeholder="E-mail" /></p>
        <p><input type="text" name="name" placeholder="Name (optional)" /></p>`}
              {(() => {
                let out = []
                this.state.selected.forEach(l => {
                  out.push(`
        <p>
            <input type="checkbox" name="l" value="${
              l.uuid
            }" id="${l.uuid.substr(0, 5)}" />
            <label for="${l.uuid.substr(0, 5)}">${l.name}</label>
        </p>`)
                })
                return out
              })()}
              {`
        <p><input type="submit" value="Subscribe" /></p>
    </div>
</form>
`}
            </pre>
          </Col>
        </Row>
      </section>
    )
  }
}

export default Forms
