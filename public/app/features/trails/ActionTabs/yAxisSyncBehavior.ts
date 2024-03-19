import { BusEventBase, BusEventWithPayload } from '@grafana/data';
import { FieldConfigBuilders, SceneDataProvider, SceneStatelessBehavior, VizPanel, sceneGraph } from '@grafana/scenes';

import { BreakdownScene } from './BreakdownScene';
import { findSceneObjectsByType } from './utils';

export class BreakdownAxisChangeEvent extends BusEventWithPayload<{ min: number; max: number }> {
  public static type = 'selected-metric-query-results-event';
}

export class ResetBreakdownAxisEvent extends BusEventBase {
  public static type = 'reset-breakdown-axis-event';
}

export const yAxisSyncBehavior: SceneStatelessBehavior = (sceneObject) => {
  const breakdownScene = sceneGraph.getAncestor(sceneObject, BreakdownScene);

  // Handle query runners from vizPanels that haven't been activated yet
  findSceneObjectsByType(sceneObject, VizPanel).forEach((vizPanel) => {
    if (vizPanel.isActive) {
      registerDataProvider(vizPanel.state.$data);
    } else {
      vizPanel.addActivationHandler(() => {
        registerDataProvider(vizPanel.state.$data);
      });
    }
  });

  // Register the data providers of all present vizpanels
  findSceneObjectsByType(sceneObject, VizPanel).forEach(registerDataProvider);

  function registerDataProvider(dataProvider?: SceneDataProvider) {
    if (!dataProvider) {
      return;
    }

    if (!dataProvider.isActive) {
      dataProvider.addActivationHandler(() => {
        // Call this function again when the dataprovider is activated
        registerDataProvider(dataProvider);
      });
    }

    // Report the panel data if it is already populated
    if (dataProvider.state.data) {
      breakdownScene.reportBreakdownPanelData(dataProvider.state.data);
    }

    // Report the panel data whenever it is updated
    const stateSubscription = dataProvider.subscribeToState((newState, prevState) => {
      if (!dataProvider.isActive) {
        stateSubscription.unsubscribe();
        return;
      }
      breakdownScene.reportBreakdownPanelData(newState.data);
    });

    // Report the panel data when there is a ResetBreakdownAxisEvent
    const axisResetSubscription = breakdownScene.subscribeToEvent(ResetBreakdownAxisEvent, (event) => {
      if (!dataProvider.isActive) {
        axisResetSubscription.unsubscribe();
        return;
      }
      breakdownScene.reportBreakdownPanelData(dataProvider.state.data);
    });
  }

  const axisChangeSubscription = breakdownScene.subscribeToEvent(BreakdownAxisChangeEvent, (event) => {
    if (!sceneObject.isActive) {
      axisChangeSubscription.unsubscribe();
      return;
    }

    const fieldConfig = FieldConfigBuilders.timeseries()
      .setCustomFieldConfig('axisSoftMin', event.payload.min)
      .setCustomFieldConfig('axisSoftMax', event.payload.max)
      .build();

    findSceneObjectsByType(sceneObject, VizPanel).forEach((vizPanel) => {
      function update() {
        vizPanel.onFieldConfigChange(fieldConfig);
        vizPanel.onOptionsChange({}); // Required to reliably ensure a refresh with the new axis range
      }

      if (vizPanel.isActive) {
        // Update axis for panels that are already active
        update();
      } else {
        vizPanel.addActivationHandler(() => {
          // Update inactive panels once they become active.
          update();
        });
      }
    });
  });
};
