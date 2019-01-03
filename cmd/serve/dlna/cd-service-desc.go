package dlna

const contentDirectoryServiceDescription = `<?xml version="1.0"?>
<scpd xmlns="urn:schemas-upnp-org:service-1-0">
  <specVersion>
    <major>1</major>
    <minor>0</minor>
  </specVersion>
  <actionList>
    <action>
      <name>GetSearchCapabilities</name>
      <argumentList>
        <argument>
          <name>SearchCaps</name>
          <direction>out</direction>
          <relatedStateVariable>SearchCapabilities</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetSortCapabilities</name>
      <argumentList>
        <argument>
          <name>SortCaps</name>
          <direction>out</direction>
          <relatedStateVariable>SortCapabilities</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetSortExtensionCapabilities</name>
      <argumentList>
        <argument>
          <name>SortExtensionCaps</name>
          <direction>out</direction>
          <relatedStateVariable>SortExtensionCapabilities</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetFeatureList</name>
      <argumentList>
        <argument>
          <name>FeatureList</name>
          <direction>out</direction>
          <relatedStateVariable>FeatureList</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetSystemUpdateID</name>
      <argumentList>
        <argument>
          <name>Id</name>
          <direction>out</direction>
          <relatedStateVariable>SystemUpdateID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>Browse</name>
      <argumentList>
        <argument>
          <name>ObjectID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>BrowseFlag</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_BrowseFlag</relatedStateVariable>
        </argument>
        <argument>
          <name>Filter</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Filter</relatedStateVariable>
        </argument>
        <argument>
          <name>StartingIndex</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Index</relatedStateVariable>
        </argument>
        <argument>
          <name>RequestedCount</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Count</relatedStateVariable>
        </argument>
        <argument>
          <name>SortCriteria</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_SortCriteria</relatedStateVariable>
        </argument>
        <argument>
          <name>Result</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Result</relatedStateVariable>
        </argument>
        <argument>
          <name>NumberReturned</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Count</relatedStateVariable>
        </argument>
        <argument>
          <name>TotalMatches</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Count</relatedStateVariable>
        </argument>
        <argument>
          <name>UpdateID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_UpdateID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>Search</name>
      <argumentList>
        <argument>
          <name>ContainerID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>SearchCriteria</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_SearchCriteria</relatedStateVariable>
        </argument>
        <argument>
          <name>Filter</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Filter</relatedStateVariable>
        </argument>
        <argument>
          <name>StartingIndex</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Index</relatedStateVariable>
        </argument>
        <argument>
          <name>RequestedCount</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Count</relatedStateVariable>
        </argument>
        <argument>
          <name>SortCriteria</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_SortCriteria</relatedStateVariable>
        </argument>
        <argument>
          <name>Result</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Result</relatedStateVariable>
        </argument>
        <argument>
          <name>NumberReturned</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Count</relatedStateVariable>
        </argument>
        <argument>
          <name>TotalMatches</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Count</relatedStateVariable>
        </argument>
        <argument>
          <name>UpdateID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_UpdateID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>CreateObject</name>
      <argumentList>
        <argument>
          <name>ContainerID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>Elements</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_Result</relatedStateVariable>
        </argument>
        <argument>
          <name>ObjectID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>Result</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_Result</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>DestroyObject</name>
      <argumentList>
        <argument>
          <name>ObjectID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>UpdateObject</name>
      <argumentList>
        <argument>
          <name>ObjectID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>CurrentTagValue</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_TagValueList</relatedStateVariable>
        </argument>
        <argument>
          <name>NewTagValue</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_TagValueList</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>MoveObject</name>
      <argumentList>
        <argument>
          <name>ObjectID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>NewParentID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>NewObjectID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>ImportResource</name>
      <argumentList>
        <argument>
          <name>SourceURI</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_URI</relatedStateVariable>
        </argument>
        <argument>
          <name>DestinationURI</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_URI</relatedStateVariable>
        </argument>
        <argument>
          <name>TransferID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_TransferID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>ExportResource</name>
      <argumentList>
        <argument>
          <name>SourceURI</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_URI</relatedStateVariable>
        </argument>
        <argument>
          <name>DestinationURI</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_URI</relatedStateVariable>
        </argument>
        <argument>
          <name>TransferID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_TransferID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>StopTransferResource</name>
      <argumentList>
        <argument>
          <name>TransferID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_TransferID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>DeleteResource</name>
      <argumentList>
        <argument>
          <name>ResourceURI</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_URI</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>GetTransferProgress</name>
      <argumentList>
        <argument>
          <name>TransferID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_TransferID</relatedStateVariable>
        </argument>
        <argument>
          <name>TransferStatus</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_TransferStatus</relatedStateVariable>
        </argument>
        <argument>
          <name>TransferLength</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_TransferLength</relatedStateVariable>
        </argument>
        <argument>
          <name>TransferTotal</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_TransferTotal</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
    <action>
      <name>CreateReference</name>
      <argumentList>
        <argument>
          <name>ContainerID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>ObjectID</name>
          <direction>in</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
        <argument>
          <name>NewID</name>
          <direction>out</direction>
          <relatedStateVariable>A_ARG_TYPE_ObjectID</relatedStateVariable>
        </argument>
      </argumentList>
    </action>
  </actionList>
  <serviceStateTable>
    <stateVariable sendEvents="no">
      <name>SearchCapabilities</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>SortCapabilities</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>SortExtensionCapabilities</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="yes">
      <name>SystemUpdateID</name>
      <dataType>ui4</dataType>
    </stateVariable>
    <stateVariable sendEvents="yes">
      <name>ContainerUpdateIDs</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="yes">
      <name>TransferIDs</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>FeatureList</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_ObjectID</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Result</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_SearchCriteria</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_BrowseFlag</name>
      <dataType>string</dataType>
      <allowedValueList>
        <allowedValue>BrowseMetadata</allowedValue>
        <allowedValue>BrowseDirectChildren</allowedValue>
      </allowedValueList>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Filter</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_SortCriteria</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Index</name>
      <dataType>ui4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_Count</name>
      <dataType>ui4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_UpdateID</name>
      <dataType>ui4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_TransferID</name>
      <dataType>ui4</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_TransferStatus</name>
      <dataType>string</dataType>
      <allowedValueList>
        <allowedValue>COMPLETED</allowedValue>
        <allowedValue>ERROR</allowedValue>
        <allowedValue>IN_PROGRESS</allowedValue>
        <allowedValue>STOPPED</allowedValue>
      </allowedValueList>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_TransferLength</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_TransferTotal</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_TagValueList</name>
      <dataType>string</dataType>
    </stateVariable>
    <stateVariable sendEvents="no">
      <name>A_ARG_TYPE_URI</name>
      <dataType>uri</dataType>
    </stateVariable>
  </serviceStateTable>
</scpd>`
